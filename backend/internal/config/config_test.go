package config

import "testing"

func TestHTTPAddressEnvironmentPrecedence(t *testing.T) {
	for _, test := range []struct {
		name, address, port, want string
	}{
		{"local default", "", "", ":8080"},
		{"hosting port", "", "10000", ":10000"},
		{"explicit address", "127.0.0.1:8088", "10000", "127.0.0.1:8088"},
		{"blank address uses hosting port", "  ", " 10000 ", ":10000"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("HTTP_ADDR", test.address)
			t.Setenv("PORT", test.port)
			if got := Load().HTTPAddr; got != test.want {
				t.Fatalf("HTTPAddr = %q, want %q", got, test.want)
			}
		})
	}
}

func TestOptionalGeminiConfiguration(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GEMINI_MODEL", "")
	cfg := Load()
	if cfg.GeminiAPIKey != "" || cfg.GeminiModel != DefaultGeminiModel {
		t.Fatal("Gemini must be optional with a known default model")
	}
	t.Setenv("GEMINI_API_KEY", " test-key ")
	t.Setenv("GEMINI_MODEL", " gemini-custom ")
	cfg = Load()
	if cfg.GeminiAPIKey != "test-key" || cfg.GeminiModel != "gemini-custom" {
		t.Fatal("Gemini environment values were not trimmed")
	}
	for _, test := range []struct {
		name, key, model string
		valid            bool
	}{
		{"disabled", "", "", true},
		{"default model", "test-key", DefaultGeminiModel, true},
		{"custom model", "test-key", "gemini-3.5-flash-lite-preview", true},
		{"missing model", "test-key", "", false},
		{"model path", "test-key", "models/gemini-test", false},
		{"model parameters", "test-key", "gemini-test?key=secret", false},
		{"model whitespace", "test-key", "gemini test", false},
		{"multiline key", "test\nkey", DefaultGeminiModel, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := Config{JWTSecret: "test-secret", CookieSameSite: "lax", GeminiAPIKey: test.key, GeminiModel: test.model}
			if err := cfg.Validate(); (err == nil) != test.valid {
				t.Fatalf("validation success=%v want=%v", err == nil, test.valid)
			}
		})
	}
}
