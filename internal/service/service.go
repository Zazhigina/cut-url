package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/Zazhigina/cut-url/internal/storage"
)

var ErrInvalidURL = errors.New("invalid url")

const maxURLLength = 2048

type URLService struct {
	storage storage.Storage
}

func NewURLService(storage storage.Storage) *URLService {
	return &URLService{storage: storage}
}

func (s *URLService) CreateShortURL(ctx context.Context, originalURL string) (string, bool, error) {
	u, err := validateURL(originalURL)
	if err != nil {
		return "", false, err
	}

	code, created, err := s.storage.Save(ctx, u.String())
	if err != nil {
		return "", false, err
	}

	return u.Scheme + "://" + u.Host + "/" + code, created, nil
}

func (s *URLService) GetOriginalURL(ctx context.Context, shortURL string) (string, error) {
	return s.storage.Get(ctx, shortURL)
}

func validateURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)

	if raw == "" {
		return nil, fmt.Errorf("%w: url is empty", ErrInvalidURL)
	}
	if len(raw) > maxURLLength {
		return nil, fmt.Errorf("%w: url is longer than %d bytes", ErrInvalidURL, maxURLLength)
	}

	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidURL, err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("%w: scheme must be http or https", ErrInvalidURL)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("%w: host is empty", ErrInvalidURL)
	}

	u.Host = strings.ToLower(u.Host)
	if u.Path == "" {
		u.Path = "/"
	}

	return u, nil
}
