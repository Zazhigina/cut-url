package random

import (
	"regexp"
	"testing"
)

const iterations = 10000

var codePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{10}$`)

func TestGenerateShortURL_Format(t *testing.T) {
	for i := 0; i < iterations; i++ {
		code, err := GenerateShortURL()
		if err != nil {
			t.Fatalf("GenerateShortURL() вернул ошибку: %v", err)
		}
		if !codePattern.MatchString(code) {
			t.Fatalf("код %q не соответствует формату задания (10 символов [a-zA-Z0-9_])", code)
		}
	}
}

func TestGenerateShortURL_UsesWholeCharset(t *testing.T) {
	seen := make(map[byte]bool, len(charset))

	for i := 0; i < iterations; i++ {
		code, err := GenerateShortURL()
		if err != nil {
			t.Fatalf("GenerateShortURL() вернул ошибку: %v", err)
		}
		for j := 0; j < len(code); j++ {
			seen[code[j]] = true
		}
	}

	for i := 0; i < len(charset); i++ {
		if !seen[charset[i]] {
			t.Errorf("символ %q ни разу не выпал за %d генераций", charset[i], iterations)
		}
	}
}

func TestGenerateShortURL_Unique(t *testing.T) {
	seen := make(map[string]struct{}, iterations)

	for i := 0; i < iterations; i++ {
		code, err := GenerateShortURL()
		if err != nil {
			t.Fatalf("GenerateShortURL() вернул ошибку: %v", err)
		}
		if _, dup := seen[code]; dup {
			t.Fatalf("код %q сгенерирован дважды за %d попыток", code, iterations)
		}
		seen[code] = struct{}{}
	}
}
