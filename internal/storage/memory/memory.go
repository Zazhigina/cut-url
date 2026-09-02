package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/Zazhigina/cut-url/pkg/random"
)

// MemoryStorage - хранилище ссылок в оперативной памяти
type MemoryStorage struct {
	mu      sync.RWMutex      // Мьютекс для защиты от конкурентного доступа
	urls    map[string]string // shortURL -> originalURL
	reverse map[string]string // originalURL -> shortURL
}

// New - создаёт новое хранилище в памяти
func New() *MemoryStorage {
	return &MemoryStorage{
		urls:    make(map[string]string),
		reverse: make(map[string]string),
	}
}

// Save - сохраняет оригинальный URL и возвращает короткую ссылку
func (m *MemoryStorage) Save(ctx context.Context, originalURL string) (string, error) {
	// Блокируем доступ на запись (аналог synchronized в Java)
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. Проверяем, существует ли уже такой URL
	if short, exists := m.reverse[originalURL]; exists {
		return short, nil // Возвращаем существующую ссылку
	}

	// 2. Генерируем новую короткую ссылку (с проверкой уникальности)
	for {
		short, err := random.GenerateShortURL()
		if err != nil {
			return "", err
		}

		// Проверяем, не занята ли эта ссылка
		if _, exists := m.urls[short]; !exists {
			// Сохраняем связь
			m.urls[short] = originalURL
			m.reverse[originalURL] = short
			return short, nil
		}
		// Если занята - пробуем сгенерировать другую
	}
}

// Get - возвращает оригинальный URL по короткой ссылке
func (m *MemoryStorage) Get(ctx context.Context, shortURL string) (string, error) {
	// Блокируем доступ на чтение (можно одновременно читать)
	m.mu.RLock()
	defer m.mu.RUnlock()

	original, exists := m.urls[shortURL]
	if !exists {
		return "", fmt.Errorf("URL not found")
	}
	return original, nil
}

// Close - закрывает хранилище (для memory ничего не нужно делать)
func (m *MemoryStorage) Close() error {
	return nil
}
