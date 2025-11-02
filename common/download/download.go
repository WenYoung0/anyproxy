package download

import (
	"context"
	"io"
)

type Downloader interface {
	Download(ctx context.Context, address string) (io.ReadCloser, error)
}
