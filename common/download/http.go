package download

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/wenyoung0/anyproxy/constant"

	"github.com/qtraffics/qtfra/log"
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
	Client *http.Client

	Attempt int
	Logger  log.Logger
}

type HTTPDownloader struct {
	logger log.Logger

	client *http.Client

	attempt int
}

func NewHTTP(option *HTTPDownloaderOption) *HTTPDownloader {
	if option == nil {
		option = &HTTPDownloaderOption{}
	}
	if option.Attempt <= 0 {
		option.Attempt = 1
	}

	var client *http.Client
	if option.Client == nil {
		client = defaultHTTPClient
	} else {
		client = option.Client
	}

	return &HTTPDownloader{
		client:  client,
		attempt: option.Attempt,
	}
}

func (d *HTTPDownloader) Download(ctx context.Context, address string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}

	response, err := d.DownloadHTTP(req)
	if err != nil {
		return nil, err
	}
	return response.Body, nil
}

func (d *HTTPDownloader) DownloadHTTP(request *http.Request) (*http.Response, error) {
	return d.download(request.Context(), request)
}

func (d *HTTPDownloader) download(ctx context.Context, request *http.Request) (response *http.Response, err error) {
	if d.client.Timeout == 0 {
		d.client.Timeout = constant.DefaultHTTPClientTimeout
	}

	attempt := d.attempt
	for attempt > 0 {
		response, err = d.client.Do(request)
		if err == nil {
			return
		}
		d.logger.Info("http download failed", slog.Int("attempt", d.attempt-attempt))
		attempt--
	}

	return
}
