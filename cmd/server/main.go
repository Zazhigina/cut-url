package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Zazhigina/cut-url/internal/config"
	"github.com/Zazhigina/cut-url/internal/handler"
	"github.com/Zazhigina/cut-url/internal/service"
	"github.com/Zazhigina/cut-url/internal/storage"
	"github.com/Zazhigina/cut-url/internal/storage/memory"
	"github.com/Zazhigina/cut-url/internal/storage/postgres"
)

func main() {
	cfg := config.Load()

	// Выбираем хранилище
	var store storage.Storage

	switch cfg.StorageType {
	case "postgres":
		log.Println("Using PostgreSQL storage")
		connString := cfg.GetDBConnString()
		pgStorage, err := postgres.New(connString)
		if err != nil {
			log.Fatalf("Failed to connect to PostgreSQL: %v", err)
		}
		store = pgStorage
		defer store.Close()
	default:
		log.Println("Using in-memory storage")
		store = memory.New()
	}

	urlService := service.NewURLService(store)
	urlHandler := handler.NewURLHandler(urlService)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(5 * time.Second))

	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "URL not found", http.StatusNotFound)
	})
	methodNotAllowed := func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
	r.MethodNotAllowed(methodNotAllowed)

	r.Post("/shorten", urlHandler.CreateShortURL)
	r.Get("/{shortURL:[a-zA-Z0-9_]{10}}", urlHandler.GetOriginalURL)

	// Запускаем сервер
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		log.Printf("Server starting on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}
