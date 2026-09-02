package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Zazhigina/cut-url/pkg/random"
	_ "github.com/jackc/pgx/v5/stdlib"

	"errors"

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

	return &PostgresStorage{db: db}, nil
}

func (p *PostgresStorage) Save(ctx context.Context, originalURL string) (string, error) {
	// 1. Проверяем, существует ли уже такой URL
	var existingShort string
	err := p.db.QueryRowContext(ctx,
		"SELECT cut_url FROM cut_url.urls WHERE origin_url = $1", originalURL,
	).Scan(&existingShort)

	if err == nil {
		return existingShort, nil
	}
	if err != sql.ErrNoRows {
		return "", fmt.Errorf("failed to check existing URL: %w", err)
	}

	// 2. Генерируем новую ссылку (с проверкой уникальности).
	// Попытки ограничены: коллизия кода маловероятна, и бесконечный цикл
	// под нагрузкой хуже, чем честная ошибка.
	for attempt := 0; attempt < maxSaveAttempts; attempt++ {
		short, err := random.GenerateShortURL()
		if err != nil {
			return "", err
		}

		// Пытаемся вставить в БД
		_, err = p.db.ExecContext(ctx,
			"INSERT INTO cut_url.urls (origin_url, cut_url) VALUES ($1, $2)",
			originalURL, short,
		)

		if err == nil {
			return short, nil // Успешно вставили
		}

		switch uniqueViolationConstraint(err) {
		case originURLConstraint:
			// Тот же URL успел вставить параллельный запрос между нашими
			// SELECT и INSERT. Повтор генерации тут не поможет — забираем
			// уже существующий код.
			return p.lookupShort(ctx, originalURL)
		case cutURLConstraint:
			continue // Код занят, пробуем следующий
		}

		return "", fmt.Errorf("failed to save URL: %w", err)
	}

	return "", fmt.Errorf("failed to generate unique short URL after %d attempts", maxSaveAttempts)
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
		return "", fmt.Errorf("URL not found")
	}
	if err != nil {
		return "", fmt.Errorf("failed to get URL: %w", err)
	}

	return originalURL, nil
}

// Close - закрывает соединение с БД
func (p *PostgresStorage) Close() error {
	return p.db.Close()
}

// uniqueViolationConstraint - возвращает имя нарушенного уникального
// ограничения или пустую строку, если ошибка другого рода.
func uniqueViolationConstraint(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return pgErr.ConstraintName
	}
	return ""
}
