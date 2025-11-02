package download

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	urlpkg "net/url"

	"github.com/wenyoung0/anyproxy/common"
)

var _ Downloader = (*HTTPDownloader)(nil)

const (
	SchemeHTTP  = "http"
	SchemeHTTPS = "https"
)

var defaultHTTPClient = &http.Client{
	Transport: http.DefaultTransport,
	Timeout:   30 * time.Second,
}

type HTTPDownloaderOption struct {
	Client        *http.Client
	Filter        func(ctx context.Context, url *urlpkg.URL) bool
	RequestFilter func(req *http.Request) error
}

type HTTPDownloader struct {
	client *http.Client

	urlFilter func(ctx context.Context, url *urlpkg.URL) bool
	reqFilter func(req *http.Request) error
}

func NewHTTP(option *HTTPDownloaderOption) *HTTPDownloader {
	if option == nil {
		option = &HTTPDownloaderOption{}
	}

	var client *http.Client
	if option == nil || option.Client == nil {
		client = defaultHTTPClient
	} else {
		client = option.Client
	}
	
	return &HTTPDownloader{
		client:    client,
		urlFilter: option.Filter,
		reqFilter: option.RequestFilter,
	}
}

func (d *HTTPDownloader) Download(ctx context.Context, address string) (io.ReadCloser, error) {
	url, err := urlpkg.Parse(address)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}
	if d.urlFilter != nil && !d.urlFilter(ctx, url) {
		return nil, fmt.Errorf("filter out")
	}

	if url.Scheme == "" {
		url.Scheme = SchemeHTTPS
	} else if url.Scheme != SchemeHTTP && url.Scheme != SchemeHTTPS {
		return nil, fmt.Errorf("unspported scheme: %s", url.Scheme)
	}

	req := common.Must(http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil))
	if d.reqFilter != nil {
		err = d.reqFilter(req)
		if err != nil {
			return nil, fmt.Errorf("RequestFilter failed: %w", err)
		}
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}
