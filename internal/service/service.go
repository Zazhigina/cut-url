package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/Zazhigina/cut-url/internal/storage"
)

// ErrInvalidURL - ссылку отклонили на валидации. Хендлер отличает её от
// внутренних ошибок и отвечает 400, а не 500.
var ErrInvalidURL = errors.New("invalid url")

// maxURLLength - предел длины ссылки в байтах.
//
// Колонка origin_url имеет тип text и сама по себе длину не ограничивает, но
// по ней построен уникальный btree-индекс, а он не принимает записи длиннее
// ~2704 байт. Без предела достаточно длинная ссылка роняла бы вставку уже на
// уровне индекса; 2048 байт заведомо укладывается и с запасом перекрывает
// реальные URL.
const maxURLLength = 2048

type URLService struct {
	storage storage.Storage
}

func NewURLService(storage storage.Storage) *URLService {
	return &URLService{storage: storage}
}

func (s *URLService) CreateShortURL(ctx context.Context, originalURL string) (string, error) {
	originalURL, err := validateURL(originalURL)
	if err != nil {
		return "", err
	}

	// Сохраняем в хранилище
	shortURL, err := s.storage.Save(ctx, originalURL)
	if err != nil {
		return "", err
	}

	return shortURL, nil
}

func (s *URLService) GetOriginalURL(ctx context.Context, shortURL string) (string, error) {
	if shortURL == "" {
		return "", fmt.Errorf("short URL cannot be empty")
	}

	return s.storage.Get(ctx, shortURL)
}

// validateURL - проверяет ссылку и возвращает её в том виде, в котором она
// попадёт в хранилище.
//
// Схема ограничена http и https сознательно: сокращённая ссылка отдаётся
// клиенту заголовком Location, поэтому без этой проверки сервис соглашался бы
// редиректить на javascript:, data: и протокол-относительные //host - то есть
// работал бы как отмывалка чужих ссылок.
func validateURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)

	if raw == "" {
		return "", fmt.Errorf("%w: url is empty", ErrInvalidURL)
	}
	if len(raw) > maxURLLength {
		return "", fmt.Errorf("%w: url is longer than %d bytes", ErrInvalidURL, maxURLLength)
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrInvalidURL, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("%w: scheme must be http or https", ErrInvalidURL)
	}
	if u.Host == "" {
		return "", fmt.Errorf("%w: host is empty", ErrInvalidURL)
	}

	return raw, nil
}
