package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Zazhigina/cut-url/internal/storage"
)

const (
	selectByOrigin = "SELECT cut_url FROM cut_url.urls WHERE origin_url = $1"
	insertURL      = "INSERT INTO cut_url.urls (origin_url, cut_url) VALUES ($1, $2)"
	selectByCut    = "SELECT origin_url FROM cut_url.urls WHERE cut_url = $1"
)

const originalURL = "http://example.com/very/long/path"

var codePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{10}$`)

func newMock(t *testing.T) (*PostgresStorage, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("не удалось создать sqlmock: %v", err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("ожидания к БД не выполнены: %v", err)
		}
		db.Close()
	})

	return &PostgresStorage{db: db}, mock
}

func uniqueViolation(constraint string) error {
	return &pgconn.PgError{Code: "23505", ConstraintName: constraint}
}

func noRows() *sqlmock.Rows { return sqlmock.NewRows([]string{"cut_url"}) }

type codeCapture struct {
	codes []string
}

func (c *codeCapture) Match(v driver.Value) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	c.codes = append(c.codes, s)
	return true
}
func TestSave_ExistingURLReturnsSameCode(t *testing.T) {
	p, mock := newMock(t)

	mock.ExpectQuery(selectByOrigin).
		WithArgs(originalURL).
		WillReturnRows(noRows().AddRow("abcdefghij"))

	code, created, err := p.Save(context.Background(), originalURL)
	if err != nil {
		t.Fatalf("Save вернул ошибку: %v", err)
	}
	if code != "abcdefghij" {
		t.Errorf("код %q, ожидался %q", code, "abcdefghij")
	}
	if created {
		t.Error("created = true, ожидалось false: запись уже существовала")
	}
}

func TestSave_NewURLInserts(t *testing.T) {
	p, mock := newMock(t)

	mock.ExpectQuery(selectByOrigin).WithArgs(originalURL).WillReturnRows(noRows())
	mock.ExpectExec(insertURL).
		WithArgs(originalURL, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	code, created, err := p.Save(context.Background(), originalURL)
	if err != nil {
		t.Fatalf("Save вернул ошибку: %v", err)
	}
	if !created {
		t.Error("created = false, ожидалось true: запись создана сейчас")
	}
	if !codePattern.MatchString(code) {
		t.Errorf("код %q не соответствует формату задания", code)
	}
}

func TestSave_RetriesOnCodeCollision(t *testing.T) {
	p, mock := newMock(t)
	capture := &codeCapture{}

	mock.ExpectQuery(selectByOrigin).WithArgs(originalURL).WillReturnRows(noRows())
	mock.ExpectExec(insertURL).
		WithArgs(originalURL, capture).
		WillReturnError(uniqueViolation(cutURLConstraint))
	mock.ExpectExec(insertURL).
		WithArgs(originalURL, capture).
		WillReturnResult(sqlmock.NewResult(1, 1))

	code, created, err := p.Save(context.Background(), originalURL)
	if err != nil {
		t.Fatalf("Save вернул ошибку: %v", err)
	}
	if !created {
		t.Error("created = false, ожидалось true")
	}

	if len(capture.codes) != 2 {
		t.Fatalf("вставка пробовалась %d раз, ожидалось 2", len(capture.codes))
	}
	if capture.codes[0] == capture.codes[1] {
		t.Errorf("после коллизии повторно использован тот же код %q", capture.codes[0])
	}
	if code != capture.codes[1] {
		t.Errorf("возвращён код %q, а вставлялся %q", code, capture.codes[1])
	}
}

func TestSave_ConcurrentInsertOfSameURL(t *testing.T) {
	p, mock := newMock(t)

	mock.ExpectQuery(selectByOrigin).WithArgs(originalURL).WillReturnRows(noRows())
	mock.ExpectExec(insertURL).
		WithArgs(originalURL, sqlmock.AnyArg()).
		WillReturnError(uniqueViolation(originURLConstraint))
	mock.ExpectQuery(selectByOrigin).
		WithArgs(originalURL).
		WillReturnRows(noRows().AddRow("winnerCode"))

	code, created, err := p.Save(context.Background(), originalURL)
	if err != nil {
		t.Fatalf("Save вернул ошибку: %v", err)
	}
	if code != "winnerCode" {
		t.Errorf("код %q, ожидался %q - тот, что успел записать конкурент", code, "winnerCode")
	}
	if created {
		t.Error("created = true, ожидалось false: запись создали не мы")
	}
}

func TestSave_GivesUpAfterMaxAttempts(t *testing.T) {
	p, mock := newMock(t)

	mock.ExpectQuery(selectByOrigin).WithArgs(originalURL).WillReturnRows(noRows())
	for i := 0; i < maxSaveAttempts; i++ {
		mock.ExpectExec(insertURL).
			WithArgs(originalURL, sqlmock.AnyArg()).
			WillReturnError(uniqueViolation(cutURLConstraint))
	}

	_, created, err := p.Save(context.Background(), originalURL)
	if err == nil {
		t.Fatal("Save вернул nil, ожидалась ошибка после исчерпания попыток")
	}
	if created {
		t.Error("created = true при ошибке")
	}
	if want := fmt.Sprintf("%d attempts", maxSaveAttempts); !strings.Contains(err.Error(), want) {
		t.Errorf("текст ошибки %q не сообщает о числе попыток (%q)", err, want)
	}
}

func TestSave_SelectFails(t *testing.T) {
	p, mock := newMock(t)

	dbErr := errors.New("connection reset by peer")
	mock.ExpectQuery(selectByOrigin).WithArgs(originalURL).WillReturnError(dbErr)

	_, _, err := p.Save(context.Background(), originalURL)
	if err == nil {
		t.Fatal("Save вернул nil, ожидалась ошибка")
	}
	if !errors.Is(err, dbErr) {
		t.Errorf("исходная ошибка БД потеряна при оборачивании: %v", err)
	}
}

func TestSave_InsertFailsWithUnrelatedError(t *testing.T) {
	p, mock := newMock(t)

	dbErr := errors.New("no space left on device")
	mock.ExpectQuery(selectByOrigin).WithArgs(originalURL).WillReturnRows(noRows())
	mock.ExpectExec(insertURL).
		WithArgs(originalURL, sqlmock.AnyArg()).
		WillReturnError(dbErr)

	_, _, err := p.Save(context.Background(), originalURL)
	if err == nil {
		t.Fatal("Save вернул nil, ожидалась ошибка")
	}
	if !errors.Is(err, dbErr) {
		t.Errorf("исходная ошибка БД потеряна при оборачивании: %v", err)
	}
}

func TestGet_NotFoundReturnsSentinel(t *testing.T) {
	p, mock := newMock(t)

	mock.ExpectQuery(selectByCut).
		WithArgs("abcdefghij").
		WillReturnRows(sqlmock.NewRows([]string{"origin_url"}))

	_, err := p.Get(context.Background(), "abcdefghij")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Get вернул %v, ожидался storage.ErrNotFound", err)
	}
}

func TestGet_QueryFailsIsNotNotFound(t *testing.T) {
	p, mock := newMock(t)

	dbErr := errors.New("server closed the connection unexpectedly")
	mock.ExpectQuery(selectByCut).WithArgs("abcdefghij").WillReturnError(dbErr)

	_, err := p.Get(context.Background(), "abcdefghij")
	if err == nil {
		t.Fatal("Get вернул nil, ожидалась ошибка")
	}
	if errors.Is(err, storage.ErrNotFound) {
		t.Error("отказ БД выдан за storage.ErrNotFound - клиент получит 404 вместо 500")
	}
	if !errors.Is(err, dbErr) {
		t.Errorf("исходная ошибка БД потеряна при оборачивании: %v", err)
	}
}

func TestClose(t *testing.T) {
	p, mock := newMock(t)

	mock.ExpectClose()

	if err := p.Close(); err != nil {
		t.Errorf("Close вернул ошибку: %v", err)
	}
}

func TestUniqueViolationConstraint(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "nil", err: nil, want: ""},
		{name: "обычная ошибка", err: errors.New("boom"), want: ""},
		{name: "ошибка драйвера без строк", err: sql.ErrNoRows, want: ""},
		{
			name: "другой код PostgreSQL",
			err:  &pgconn.PgError{Code: "23503", ConstraintName: "urls_origin_url_key"},
			want: "",
		},
		{
			name: "нарушение уникальности кода",
			err:  uniqueViolation(cutURLConstraint),
			want: cutURLConstraint,
		},
		{
			name: "нарушение уникальности URL",
			err:  uniqueViolation(originURLConstraint),
			want: originURLConstraint,
		},
		{
			name: "обернутая ошибка",
			err:  fmt.Errorf("insert failed: %w", uniqueViolation(cutURLConstraint)),
			want: cutURLConstraint,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := uniqueViolationConstraint(tt.err); got != tt.want {
				t.Errorf("uniqueViolationConstraint(%v) = %q, ожидалось %q", tt.err, got, tt.want)
			}
		})
	}
}
