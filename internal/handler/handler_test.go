package handler

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/Zazhigina/cut-url/internal/service"
)

var errStorageDown = errors.New("dial tcp 10.0.0.5:5432: connection refused")

type failingStorage struct{}

func (failingStorage) Save(context.Context, string) (string, bool, error) {
	return "", false, errStorageDown
}

func (failingStorage) Get(context.Context, string) (string, error) {
	return "", errStorageDown
}

func (failingStorage) Close() error { return nil }

func newFailingHandler() *URLHandler {
	return NewURLHandler(service.NewURLService(failingStorage{}))
}

func muteLog(t *testing.T) {
	t.Helper()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
}

func TestCreateShortURL_StorageFailure(t *testing.T) {
	muteLog(t)

	req := httptest.NewRequest(http.MethodPost, "/shorten",
		strings.NewReader(`{"url":"http://example.com/very/long/path"}`))
	rec := httptest.NewRecorder()

	newFailingHandler().CreateShortURL(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("статус %d, ожидался %d: отказ хранилища - не вина клиента",
			rec.Code, http.StatusInternalServerError)
	}
	if body := rec.Body.String(); strings.Contains(body, errStorageDown.Error()) {
		t.Errorf("внутренняя ошибка утекла в тело ответа: %q", body)
	}
}

func TestGetOriginalURL_StorageFailure(t *testing.T) {
	muteLog(t)

	const code = "abcdefghij"

	req := httptest.NewRequest(http.MethodGet, "/"+code, nil)
	rec := httptest.NewRecorder()

	newFailingHandler().GetOriginalURL(rec, withURLParam(req, code))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("статус %d, ожидался %d: отказ хранилища нельзя отдавать как 404",
			rec.Code, http.StatusInternalServerError)
	}
	if body := rec.Body.String(); strings.Contains(body, errStorageDown.Error()) {
		t.Errorf("внутренняя ошибка утекла в тело ответа: %q", body)
	}
}

func withURLParam(r *http.Request, shortURL string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("shortURL", shortURL)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}
