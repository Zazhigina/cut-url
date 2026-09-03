package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Zazhigina/cut-url/internal/storage"
	"github.com/Zazhigina/cut-url/migrations"
	"github.com/Zazhigina/cut-url/pkg/random"
	_ "github.com/jackc/pgx/v5/stdlib"

	"errors"
	"log"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	originURLConstraint = "urls_origin_url_key"
	cutURLConstraint    = "urls_cut_url_key"

	maxSaveAttempts = 10
)

type PostgresStorage struct {
	db *sql.DB
}

func New(connString string) (*PostgresStorage, error) {
	db, err := sql.Open("pgx", connString)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	if _, err := db.ExecContext(ctx, migrations.CreateURLsTable); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to apply migration: %w", err)
	}
	log.Println("Schema is up to date")

	return &PostgresStorage{db: db}, nil
}

func (p *PostgresStorage) Save(ctx context.Context, originalURL string) (string, bool, error) {
	// 1. Проверяем, существует ли уже такой URL
	var existingShort string
	err := p.db.QueryRowContext(ctx,
		"SELECT cut_url FROM cut_url.urls WHERE origin_url = $1", originalURL,
	).Scan(&existingShort)

	if err == nil {
		return existingShort, false, nil
	}
	if err != sql.ErrNoRows {
		return "", false, fmt.Errorf("failed to check existing URL: %w", err)
	}

	for attempt := 0; attempt < maxSaveAttempts; attempt++ {
		short, err := random.GenerateShortURL()
		if err != nil {
			return "", false, err
		}

		_, err = p.db.ExecContext(ctx,
			"INSERT INTO cut_url.urls (origin_url, cut_url) VALUES ($1, $2)",
			originalURL, short,
		)

		if err == nil {
			return short, true, nil
		}

		switch uniqueViolationConstraint(err) {
		case originURLConstraint:
			short, err := p.lookupShort(ctx, originalURL)
			return short, false, err
		case cutURLConstraint:
			continue
		}

		return "", false, fmt.Errorf("failed to save URL: %w", err)
	}

	return "", false, fmt.Errorf("failed to generate unique short URL after %d attempts", maxSaveAttempts)
}

// lookupShort - возвращает уже сохранённый код для оригинального URL
func (p *PostgresStorage) lookupShort(ctx context.Context, originalURL string) (string, error) {
	var short string
	err := p.db.QueryRowContext(ctx,
		"SELECT cut_url FROM cut_url.urls WHERE origin_url = $1", originalURL,
	).Scan(&short)
	if err != nil {
		return "", fmt.Errorf("failed to get existing URL: %w", err)
	}
	return short, nil
}

// Get - возвращает оригинальный URL по короткой ссылке
func (p *PostgresStorage) Get(ctx context.Context, shortURL string) (string, error) {
	var originalURL string
	err := p.db.QueryRowContext(ctx,
		"SELECT origin_url FROM cut_url.urls WHERE cut_url = $1", shortURL,
	).Scan(&originalURL)

	if err == sql.ErrNoRows {
		return "", storage.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("failed to get URL: %w", err)
	}

	return originalURL, nil
}

func (p *PostgresStorage) Close() error {
	return p.db.Close()
}

func uniqueViolationConstraint(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return pgErr.ConstraintName
	}
	return ""
}
