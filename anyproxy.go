package anyproxy

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wenyoung0/anyproxy/common/download"
	"github.com/wenyoung0/anyproxy/config"
	"github.com/wenyoung0/anyproxy/constant"

	"github.com/qtraffics/qtfra/enhancements/iolib"
	"github.com/qtraffics/qtfra/ex"
	"github.com/qtraffics/qtfra/log"
)

var errExitSuccess = errors.New("exit success")

type Proxy struct {
	allowance      map[string]bool
	httpDownloader *download.HTTPDownloader

	config config.Config
}

func NewProxy(c config.Config) (*Proxy, error) {
	if len(c.Allowance) == 0 {
		return nil, fmt.Errorf("no allowance set")
	}

	allowance := make(map[string]bool)
	for _, v := range c.Allowance {
		allowance[v] = true
	}

	customClient := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: constant.DefaultHTTPClientTimeout,
			}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 5 * time.Minute,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	p := &Proxy{
		httpDownloader: download.NewHTTP(&download.HTTPDownloaderOption{
			Client: customClient,
		}),
		config:    c,
		allowance: allowance,
	}

	return p, nil
}

func (p *Proxy) Serve(wg *sync.WaitGroup, ctx context.Context) error {
	cancelCtx, cancel := context.WithCancelCause(ctx)

	logger := log.GetDefaultLogger()

	httpServer := &http.Server{}
	httpServer.Handler = p.Handler(cancelCtx)
	httpServer.Addr = net.JoinHostPort(p.config.Listen, strconv.FormatUint(uint64(p.config.Port), 10))
	httpServer.BaseContext = func(_ net.Listener) context.Context { return cancelCtx }

	listener, err := net.Listen("tcp", httpServer.Addr)
	if err != nil {
		cancel(errExitSuccess)
		return err
	}

	wg.Go(func() {
		defer listener.Close()
		logger.Info("Server started", slog.String("address", listener.Addr().String()))
		err := httpServer.Serve(listener)
		if err != nil {
			logger.Warn("Server closed with error", log.AttrError(err))
			if !ex.IsMulti(err, net.ErrClosed) {
				cancel(err)
				return
			}

		}
		logger.Warn("Server closed")
		cancel(errExitSuccess)
	})

	wg.Go(func() {
		var err error
		<-cancelCtx.Done()

		timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer timeoutCancel()
		err = httpServer.Shutdown(timeoutCtx)
		causeErr := context.Cause(cancelCtx)
		if errors.Is(causeErr, errExitSuccess) {
			causeErr = nil
		}
		err = errors.Join(err, causeErr)
		if err != nil {
			logger.Error("close server error", log.AttrError(err))
		}
	})

	return nil
}

func (p *Proxy) Handler(ctx context.Context) http.Handler {
	logger := log.GetDefaultLogger()
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		//if req.Method != http.MethodGet && req.Method != http.MethodHead {
		//	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Allow
		//	resp.Header().Add("Allow", http.MethodGet)
		//	resp.Header().Add("Allow", http.MethodHead)
		//
		//	resp.WriteHeader(http.StatusMethodNotAllowed)
		//	return
		//}

		target := cmp.Or(req.URL.RawPath, req.URL.Path)
		target = strings.TrimLeft(target, "/")
		if len(target) == 0 {
			resp.WriteHeader(http.StatusNotFound)
			return
		}

		if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
			target = "https://" + target
		}
		if len(req.URL.RawQuery) != 0 {
			target += "?" + req.URL.RawQuery
		}

		logger.Info("Start new task for",
			slog.String("target", target),
			slog.String("method", req.Method))
		p.startNewTask(req.Context(), resp, req, target)
	})
}

func (p *Proxy) startNewTask(ctx context.Context, clientResponse http.ResponseWriter, clientRequest *http.Request, target string) {
	logger := log.GetDefaultLogger()

	request, err := http.NewRequestWithContext(ctx, clientRequest.Method, target, clientRequest.Body)
	if err != nil {
		http.Error(clientResponse, err.Error(), http.StatusInternalServerError)
		return
	}

	if !p.allowance[request.URL.Hostname()] {
		http.Error(clientResponse, "Access denied.", http.StatusForbidden)
		return
	}

	maps.Copy(request.Header, clientRequest.Header)

	logger.Debug("Start download", slog.String("target", target))
	response, err := p.httpDownloader.DownloadHTTP(request)
	if err != nil {
		logger.Info("Download failed!", slog.String("target", target), log.AttrError(err))
		http.Error(clientResponse, err.Error(), http.StatusInternalServerError)
		return
	}

	defer response.Body.Close()

	// Copy headers
	for k, v := range response.Header {
		for _, vv := range v {
			clientResponse.Header().Add(k, vv)
		}
	}

	clientResponse.WriteHeader(response.StatusCode)

	// Copy the response body to the client
	if _, err := iolib.Copy(response.Body, clientResponse); err != nil {
		logger.Error("Error copying response body", log.AttrError(err))
	}

	logger.Debug("Copy finished")
}
