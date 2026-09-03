package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Zazhigina/cut-url/internal/model"
	"github.com/Zazhigina/cut-url/internal/service"
	"github.com/Zazhigina/cut-url/internal/storage"
)

type URLHandler struct {
	service *service.URLService
}

func NewURLHandler(service *service.URLService) *URLHandler {
	return &URLHandler{service: service}
}

func (h *URLHandler) CreateShortURL(w http.ResponseWriter, r *http.Request) {
	var req model.CreateURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	shortURL, created, err := h.service.CreateShortURL(r.Context(), req.URL)
	if err != nil {
		if errors.Is(err, service.ErrInvalidURL) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Printf("create short url: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	resp := model.CreateURLResponse{ShortURL: shortURL, Created: created}
	status := http.StatusCreated
	if !created {
		status = http.StatusOK
		resp.Message = "URL has already been shortened"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

func (h *URLHandler) GetOriginalURL(w http.ResponseWriter, r *http.Request) {
	shortURL := chi.URLParam(r, "shortURL")

	originalURL, err := h.service.GetOriginalURL(r.Context(), shortURL)
	if err != nil {

		if errors.Is(err, storage.ErrNotFound) {
			http.Error(w, "URL not found", http.StatusNotFound)
			return
		}
		log.Printf("get original url: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model.GetURLResponse{URL: originalURL})
}
