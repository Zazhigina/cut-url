package memory

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

const goroutines = 100

func TestSave_SameURLConcurrently(t *testing.T) {
	const originalURL = "http://example.com/very/long/path"

	m := New()
	ctx := context.Background()

	codes := make([]string, goroutines)
	createdFlags := make([]bool, goroutines)
	errs := make([]error, goroutines)

	start := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			codes[i], createdFlags[i], errs[i] = m.Save(ctx, originalURL)
		}(i)
	}

	close(start)
	wg.Wait()

	createdCount := 0
	for i := 0; i < goroutines; i++ {
		if errs[i] != nil {
			t.Fatalf("горутина %d: Save вернул ошибку: %v", i, errs[i])
		}
		if codes[i] != codes[0] {
			t.Fatalf("горутина %d получила код %q, а горутина 0 - %q: на один URL выдано несколько ссылок",
				i, codes[i], codes[0])
		}
		if createdFlags[i] {
			createdCount++
		}
	}

	if createdCount != 1 {
		t.Errorf("created = true вернулся %d раз, ожидался ровно 1: запись создаётся однократно", createdCount)
	}

	got, err := m.Get(ctx, codes[0])
	if err != nil {
		t.Fatalf("Get(%q) вернул ошибку: %v", codes[0], err)
	}
	if got != originalURL {
		t.Errorf("Get вернул %q, ожидалось %q", got, originalURL)
	}
}

func TestSave_StorageIsNotEmpty(t *testing.T) {
	m := New()
	ctx := context.Background()

	if len(m.urls) != 0 {
		t.Fatalf("новое хранилище не пустое: %d записей", len(m.urls))
	}

	if _, _, err := m.Save(ctx, "http://example.com/some/path"); err != nil {
		t.Fatalf("Save вернул ошибку: %v", err)
	}

	if len(m.urls) == 0 {
		t.Fatal("после Save хранилище пустое, запись не сохранилась")
	}
	if len(m.reverse) != len(m.urls) {
		t.Fatalf("reverse (%d) и urls (%d) рассинхронизированы", len(m.reverse), len(m.urls))
	}
}

func TestSave_DifferentURLsConcurrently(t *testing.T) {
	m := New()
	ctx := context.Background()

	urls := make([]string, goroutines)
	codes := make([]string, goroutines)
	errs := make([]error, goroutines)
	for i := range urls {
		urls[i] = fmt.Sprintf("http://example.com/page-%d", i)
	}

	start := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			codes[i], _, errs[i] = m.Save(ctx, urls[i])
		}(i)
	}

	close(start)
	wg.Wait()

	seen := make(map[string]string, goroutines)
	for i := 0; i < goroutines; i++ {
		if errs[i] != nil {
			t.Fatalf("горутина %d: Save вернул ошибку: %v", i, errs[i])
		}
		if owner, dup := seen[codes[i]]; dup {
			t.Fatalf("код %q выдан и для %q, и для %q", codes[i], owner, urls[i])
		}
		seen[codes[i]] = urls[i]
	}

	if len(seen) != goroutines {
		t.Fatalf("сохранено %d уникальных кодов, ожидалось %d", len(seen), goroutines)
	}

	for code, wantURL := range seen {
		got, err := m.Get(ctx, code)
		if err != nil {
			t.Fatalf("Get(%q) вернул ошибку: %v", code, err)
		}
		if got != wantURL {
			t.Errorf("Get(%q) вернул %q, ожидалось %q", code, got, wantURL)
		}
	}
}

func TestSaveAndGetConcurrently(t *testing.T) {
	m := New()
	ctx := context.Background()

	code, _, err := m.Save(ctx, "http://example.com/first")
	if err != nil {
		t.Fatalf("подготовка: Save вернул ошибку: %v", err)
	}

	start := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			<-start
			if _, _, err := m.Save(ctx, fmt.Sprintf("http://example.com/writer-%d", i)); err != nil {
				t.Errorf("Save вернул ошибку: %v", err)
			}
		}(i)
		go func() {
			defer wg.Done()
			<-start
			if _, err := m.Get(ctx, code); err != nil {
				t.Errorf("Get(%q) вернул ошибку: %v", code, err)
			}
		}()
	}

	close(start)
	wg.Wait()
}
