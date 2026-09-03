package storage

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("url not found")

type Storage interface {
	Save(ctx context.Context, originalURL string) (short string, created bool, err error)
	Get(ctx context.Context, shortURL string) (string, error)
	Close() error
}
