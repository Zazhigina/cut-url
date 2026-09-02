package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Zazhigina/cut-url/internal/config"
	"github.com/Zazhigina/cut-url/internal/handler"
	"github.com/Zazhigina/cut-url/internal/service"
	"github.com/Zazhigina/cut-url/internal/storage/memory"
	"github.com/Zazhigina/cut-url/internal/storage/postgres"
)

func main() {
	cfg := config.Load()

	// Выбираем хранилище
	var storage interface {
		Save(ctx context.Context, originalURL string) (string, error)
		Get(ctx context.Context, shortURL string) (string, error)
		Close() error
	}

	switch cfg.StorageType {
	case "postgres":
		log.Println("Using PostgreSQL storage")
		connString := cfg.GetDBConnString()
		pgStorage, err := postgres.New(connString)
		if err != nil {
			log.Fatalf("Failed to connect to PostgreSQL: %v", err)
		}
		storage = pgStorage
		defer storage.Close()
	default:
		log.Println("Using in-memory storage")
		storage = memory.New()
	}

	// Создаём сервис и хендлер
	urlService := service.NewURLService(storage)
	urlHandler := handler.NewURLHandler(urlService)

	// Регистрируем маршруты
	http.HandleFunc("/shorten", urlHandler.CreateShortURL)
	http.HandleFunc("/", urlHandler.GetOriginalURL)

	// Запускаем сервер
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: nil,
	}

	// Graceful shutdown (корректное завершение)
	go func() {
		log.Printf("Server starting on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Ожидаем сигнал остановки
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Даём время завершить текущие запросы
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}
