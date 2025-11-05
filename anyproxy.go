package anyproxy

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	urlpkg "net/url"
	"strconv"
	"strings"
	"time"

	"github.com/qtraffics/qtfra/enhancements/iolib"
	"github.com/wenyoung0/anyproxy/common/download"
	"github.com/wenyoung0/anyproxy/config"
	"github.com/wenyoung0/anyproxy/constant"

	"github.com/qtraffics/qtfra/ex"
	"github.com/qtraffics/qtfra/log"
)

var errExitSuccess = errors.New("exit success")

type Proxy struct {
	allowance map[string]bool

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

	// Custom client with timeouts
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
			URLFilter: func(ctx context.Context, url *urlpkg.URL) bool {
				return allowance[url.Hostname()]
			},
		}),
		config: c,
	}

	return p, nil
}

func (p *Proxy) Serve(ctx context.Context) error {
	cancelCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(errExitSuccess)

	httpServer := &http.Server{}
	httpServer.Handler = p.Handler(cancelCtx)
	httpServer.Addr = net.JoinHostPort(p.config.Listen, strconv.FormatUint(uint64(p.config.Port), 10))
	httpServer.BaseContext = func(_ net.Listener) context.Context { return cancelCtx }

	go func() {
		log.GetDefaultLogger().Info("Server started", slog.String("address", httpServer.Addr))
		err := httpServer.ListenAndServe()

		if err != nil && !ex.IsMulti(err, net.ErrClosed) {
			cancel(err)
		}
		cancel(errExitSuccess)
	}()

	var err error
	select {
	case <-cancelCtx.Done():
		timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer timeoutCancel()
		err = httpServer.Shutdown(timeoutCtx)
		causeErr := context.Cause(cancelCtx)
		if errors.Is(causeErr, errExitSuccess) {
			causeErr = nil
		}
		err = errors.Join(err, causeErr)
	}
	return err
}

func (p *Proxy) Handler(ctx context.Context) http.Handler {
	logger := log.GetDefaultLogger()
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			// https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Allow
			resp.Header().Add("Allow", http.MethodGet)
			resp.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		target := cmp.Or(req.URL.RawPath, req.URL.Path)
		target = strings.TrimLeft(target, "/")
		if len(target) == 0 {
			resp.WriteHeader(http.StatusNotFound)
			return
		}

		logger.Info("Start new task for", slog.String("target", target))
		p.startNewTask(req.Context(), resp, req, target)
	})
}

func (p *Proxy) startNewTask(ctx context.Context, resp http.ResponseWriter, req *http.Request, target string) {
	logger := log.GetDefaultLogger()

	targetReq, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}

	p.copyHeader(req.Header, targetReq.Header)

	logger.Debug("Start download", slog.String("target", target))
	targetResponse, err := p.httpDownloader.DownloadHTTP(targetReq)
	if err != nil {
		logger.Info("Download failed!", slog.String("target", target), log.AttrError(err))
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}

	defer targetResponse.Body.Close()

	// Copy headers
	for k, vv := range targetResponse.Header {
		for _, v := range vv {
			resp.Header().Add(k, v)
		}
	}
	resp.WriteHeader(targetResponse.StatusCode)

	// Copy the response body to the client
	if _, err := iolib.Copy(targetResponse.Body, resp); err != nil {
		logger.Error("Error copying response body", log.AttrError(err))
	}
	logger.Debug("Copy finished")
}

func (p *Proxy) copyHeader(source, destination http.Header) {
	mapping := map[string]string{
		"User-Agent":        source.Get("User-Agent"),
		"Cookie":            source.Get("Cookie"),
		"Accept":            source.Get("Accept"),
		"Accept-Encoding":   source.Get("Accept-Encoding"),
		"Accept-Language":   source.Get("Accept-Language"),
		"Referer":           source.Get("Referer"), // optional
		"Authorization":     source.Get("Authorization"),
		"If-Modified-Since": source.Get("If-Modified-Since"),
		"If-None-Match":     source.Get("If-None-Match"),
		"Range":             source.Get("Range"),
		"Origin":            source.Get("Origin"),
	}

	for header, headerValue := range mapping {
		if len(headerValue) == 0 {
			continue
		}
		destination.Set(header, headerValue)
	}
}
