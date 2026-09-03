package service

import (
	"context"
	"testing"
)

const stubCode = "abcdefghij"

type captureStorage struct {
	saved string
}

func (c *captureStorage) Save(_ context.Context, originalURL string) (string, bool, error) {
	c.saved = originalURL
	return stubCode, true, nil
}

func (c *captureStorage) Get(context.Context, string) (string, error) { return "", nil }

func (c *captureStorage) Close() error { return nil }

func TestCreateShortURL_NormalizationAndShortLink(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantSaved string
		wantShort string
	}{
		{
			name:      "пробелы по краям обрезаются",
			input:     "   http://example.com/very/long/path   ",
			wantSaved: "http://example.com/very/long/path",
			wantShort: "http://example.com/" + stubCode,
		},
		{
			name:      "схема и хост приводятся к нижнему регистру, путь - нет",
			input:     "HTTP://EXAMPLE.COM/Very/Long/Path",
			wantSaved: "http://example.com/Very/Long/Path",
			wantShort: "http://example.com/" + stubCode,
		},
		{
			name:      "пустой путь становится слешем",
			input:     "http://example.com",
			wantSaved: "http://example.com/",
			wantShort: "http://example.com/" + stubCode,
		},
		{
			name:      "query и якорь хранятся целиком, но в короткую ссылку не попадают",
			input:     "https://example.com/search?q=go+lang&page=2#results",
			wantSaved: "https://example.com/search?q=go+lang&page=2#results",
			wantShort: "https://example.com/" + stubCode,
		},
		{
			name:      "порт исходного адреса сохраняется",
			input:     "http://example.com:8443/a/b/c",
			wantSaved: "http://example.com:8443/a/b/c",
			wantShort: "http://example.com:8443/" + stubCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &captureStorage{}
			svc := NewURLService(store)

			short, created, err := svc.CreateShortURL(context.Background(), tt.input)
			if err != nil {
				t.Fatalf("CreateShortURL(%q) вернул ошибку: %v", tt.input, err)
			}
			if !created {
				t.Errorf("created = false, ожидалось true для новой ссылки")
			}
			if store.saved != tt.wantSaved {
				t.Errorf("в хранилище ушло %q, ожидалось %q", store.saved, tt.wantSaved)
			}
			if short != tt.wantShort {
				t.Errorf("короткая ссылка %q, ожидалось %q", short, tt.wantShort)
			}
		})
	}
}
