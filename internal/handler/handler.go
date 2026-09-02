package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/Zazhigina/cut-url/internal/model"
	"github.com/Zazhigina/cut-url/internal/service"
)

type URLHandler struct {
	service *service.URLService
}

func NewURLHandler(service *service.URLService) *URLHandler {
	return &URLHandler{service: service}
}

func (h *URLHandler) CreateShortURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req model.CreateURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	shortURL, err := h.service.CreateShortURL(r.Context(), req.URL)
	if err != nil {
		if errors.Is(err, service.ErrInvalidURL) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Printf("create short url: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	resp := model.CreateURLResponse{ShortURL: shortURL}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *URLHandler) GetOriginalURL(w http.ResponseWriter, r *http.Request) {
	shortURL := r.URL.Path[1:]
	if shortURL == "" {
		http.Error(w, "Short URL is required", http.StatusBadRequest)
		return
	}

	originalURL, err := h.service.GetOriginalURL(r.Context(), shortURL)
	if err != nil {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusFound) // 302
}
