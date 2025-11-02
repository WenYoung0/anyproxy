package anyproxy

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/wenyoung0/anyproxy/common/download"
	"github.com/wenyoung0/anyproxy/config"
)

var errExitSuccess = errors.New("exit success")

type Proxy struct {
	server    *http.Server
	listener  net.Listener
	allowance map[string]struct{}

	download download.Downloader
}

func NewProxy(c config.Config) (*Proxy, error) {
	var p = new(Proxy)
	var err error
	p.listener, err = net.Listen("tcp", net.JoinHostPort(c.Listen, strconv.FormatUint(uint64(c.Port), 10)))
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}
	if len(c.Allowance) == 0 {
		return nil, fmt.Errorf("no allowance set")
	}
	p.allowance = make(map[string]struct{})
	for _, v := range c.Allowance {
		p.allowance[v] = struct{}{}
	}
	return p, nil
}

func (p *Proxy) Serve(ctx context.Context) error {
	cancelCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(errExitSuccess)
	go func() {
		err := p.server.Serve(p.listener)
		if err != nil {
			cancel(err)
		}
		cancel(errExitSuccess)
	}()

	var err error
	select {
	case <-cancelCtx.Done():
		timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer timeoutCancel()
		err = p.server.Shutdown(timeoutCtx)
		causeErr := context.Cause(cancelCtx)
		if errors.Is(causeErr, errExitSuccess) {
			causeErr = nil
		}
		err = errors.Join(err, causeErr)
	}
	return err
}

func (p *Proxy) Handler() http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			resp.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		target := cmp.Or(req.URL.RawPath, req.URL.Path)
		target = strings.TrimSpace(target)
		if len(target) == 0 {
			resp.WriteHeader(http.StatusNoContent)
			return
		}

	})
}

func (p *Proxy) copyConn(ctx context.Context, source io.Writer, address string) (int64, error) {
	reader, err := p.download.Download(ctx, address)
	if err != nil {
		return 0, err
	}
	written, err := io.Copy(source, reader)

}
