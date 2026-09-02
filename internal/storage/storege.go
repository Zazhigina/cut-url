package storage

import "context"

type Storage interface {
	Save(ctx context.Context, originalURL string) (string, error)
	Get(ctx context.Context, shortURL string) (string, error)
	Close() error
}
