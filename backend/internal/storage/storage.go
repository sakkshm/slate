package storage

import (
	"context"
	"io"
)

type Store interface {
	Upload(context.Context, string, io.Reader, int64) error
	Download(context.Context, string) (io.ReadCloser, error)
	Exists(context.Context, string) (bool, error)
	List(context.Context, string) ([]string, error)
	Delete(context.Context, string) error
}
