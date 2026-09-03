package memory

import (
	"context"
	"sync"

	"github.com/Zazhigina/cut-url/internal/storage"
	"github.com/Zazhigina/cut-url/pkg/random"
)

const maxEntries = 100_000

type MemoryStorage struct {
	mu      sync.RWMutex
	urls    map[string]string // shortURL -> originalURL
	reverse map[string]string // originalURL -> shortURL
	order   []string
}

func New() *MemoryStorage {
	return &MemoryStorage{
		urls:    make(map[string]string),
		reverse: make(map[string]string),
	}
}

// Save - сохраняет оригинальный URL и возвращает короткую ссылку
func (m *MemoryStorage) Save(ctx context.Context, originalURL string) (string, bool, error) {
	// Блокируем доступ на запись (аналог synchronized в Java)
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. Проверяем, существует ли уже такой URL
	if short, exists := m.reverse[originalURL]; exists {
		return short, false, nil // Возвращаем существующую ссылку
	}

	// 2. Генерируем новую короткую ссылку (с проверкой уникальности)
	for {
		short, err := random.GenerateShortURL()
		if err != nil {
			return "", false, err
		}

		if _, exists := m.urls[short]; !exists {
			m.urls[short] = originalURL
			m.reverse[originalURL] = short
			m.order = append(m.order, short)

			if len(m.order) > maxEntries {
				oldest := m.order[0]
				m.order = m.order[1:]
				delete(m.reverse, m.urls[oldest])
				delete(m.urls, oldest)
			}

			return short, true, nil
		}
	}
}

// Get - возвращает оригинальный URL по короткой ссылке
func (m *MemoryStorage) Get(ctx context.Context, shortURL string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	original, exists := m.urls[shortURL]
	if !exists {
		return "", storage.ErrNotFound
	}
	return original, nil
}

func (m *MemoryStorage) Close() error {
	return nil
}
