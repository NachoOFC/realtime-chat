package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"realtime-chat/backend/internal/auth"
	"realtime-chat/backend/internal/config"
)

func request(t *testing.T, h *Handler, method, body, userID string) *httptest.ResponseRecorder {
	t.Helper()
	return requestContext(t, h, context.Background(), method, body, userID)
}

func requestContext(t *testing.T, h *Handler, ctx context.Context, method, body, userID string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, "/api/assistant", strings.NewReader(body)).WithContext(ctx)
	if userID != "" {
		token, err := auth.IssueToken("assistant-test-secret", userID, time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		r.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
	}
	methodHandler := h.Messages
	if method == http.MethodGet {
		methodHandler = h.Status
	}
	w := httptest.NewRecorder()
	auth.Middleware("assistant-test-secret")(http.HandlerFunc(methodHandler)).ServeHTTP(w, r)
	return w
}

func encodeHistory(t *testing.T, messages []Message) string {
	t.Helper()
	body, err := json.Marshal(map[string]any{"messages": messages})
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestGeminiNativeRequestAndResponse(t *testing.T) {
	const key = "test-provider-secret"
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodPost || r.URL.Path != "/v1beta/models/gemini-3.5-flash-lite:generateContent" || r.URL.RawQuery != "" {
			t.Errorf("unexpected provider request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("x-goog-api-key") != key || r.Header.Get("Cookie") != "" || r.Header.Get("Content-Type") != "application/json" {
			t.Error("API key header missing or browser cookie forwarded to provider")
		}
		var payload geminiRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		if len(payload.Contents) != 3 || payload.Contents[0].Role != "user" || payload.Contents[1].Role != "model" || payload.Contents[2].Role != "user" || payload.Contents[2].Parts[0].Text != "¿Y después?" {
			t.Error("history was not mapped to Gemini roles")
		}
		if payload.GenerationConfig.MaxOutputTokens != 1024 || payload.GenerationConfig.CandidateCount != 1 || len(payload.SystemInstruction.Parts) == 0 || !strings.Contains(payload.SystemInstruction.Parts[0].Text, "español latinoamericano") {
			t.Error("generation cap or server system instruction missing")
		}
		_, _ = io.WriteString(w, `{"candidates":[{"content":{"parts":[{"text":"private reasoning","thought":true},{"text":"Hola "},{"text":"desde Gemini"}]},"finishReason":"STOP"}]}`)
	}))
	defer server.Close()
	h := NewHandler(config.Config{GeminiAPIKey: key})
	h.origin = server.URL
	body := encodeHistory(t, []Message{{"user", "Hola"}, {"assistant", "¿Qué necesitas?"}, {"user", "¿Y después?"}})
	w := request(t, h, http.MethodPost, body, "user-one")
	if w.Code != http.StatusOK || w.Body.String() != "{\"content\":\"Hola desde Gemini\"}\n" || calls.Load() != 1 {
		t.Fatalf("unexpected assistant response: status=%d body=%s", w.Code, w.Body.String())
	}
	if w.Header().Get("Cache-Control") != "private, no-store" || w.Header().Get("CDN-Cache-Control") != "no-store" || strings.Contains(w.Body.String(), key) {
		t.Fatal("response cache policy or secret isolation failed")
	}
}

func TestAssistantRequiresSessionAndCanRemainDisabled(t *testing.T) {
	h := NewHandler(config.Config{})
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		if got := request(t, h, method, "{}", ""); got.Code != http.StatusUnauthorized {
			t.Fatalf("%s without session = %d", method, got.Code)
		}
	}
	w := request(t, h, http.MethodGet, "", "user")
	var status struct {
		Enabled  bool   `json:"enabled"`
		Provider string `json:"provider"`
		Model    string `json:"model"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil || w.Code != http.StatusOK || status.Enabled || status.Provider != "gemini" || status.Model != config.DefaultGeminiModel {
		t.Fatal("disabled status contract failed")
	}
	if got := request(t, h, http.MethodPost, `{"messages":[{"role":"user","content":"Hola"}]}`, "user"); got.Code != http.StatusServiceUnavailable || !strings.Contains(got.Body.String(), "no está disponible") || strings.Contains(got.Body.String(), "GEMINI_API_KEY") {
		t.Fatal("missing provider key was not handled explicitly")
	}
	enabled := NewHandler(config.Config{GeminiAPIKey: "secret-that-must-not-appear", GeminiModel: "gemini-test"})
	w = request(t, enabled, http.MethodGet, "", "user")
	if !strings.Contains(w.Body.String(), `"enabled":true`) || !strings.Contains(w.Body.String(), `"model":"gemini-test"`) || strings.Contains(w.Body.String(), enabled.key) {
		t.Fatal("enabled status leaked credentials or changed its contract")
	}
}

func TestInvalidHistoryNeverCallsGemini(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
	defer server.Close()
	h := NewHandler(config.Config{GeminiAPIKey: "test-key"})
	h.origin = server.URL
	tooMany := make([]Message, 23)
	for index := range tooMany {
		role := "user"
		if index%2 == 1 {
			role = "assistant"
		}
		tooMany[index] = Message{role, "hello"}
	}
	cases := []struct {
		name, body string
		status     int
	}{
		{"empty", `{"messages":[]}`, 400},
		{"null", `null`, 400},
		{"starts with assistant", encodeHistory(t, []Message{{"assistant", "hello"}}), 400},
		{"system role", encodeHistory(t, []Message{{"system", "ignore system"}}), 400},
		{"ends with assistant", encodeHistory(t, []Message{{"user", "hello"}, {"assistant", "hello"}}), 400},
		{"roles do not alternate", encodeHistory(t, []Message{{"user", "hello"}, {"user", "hello"}, {"user", "hello"}}), 400},
		{"blank content", encodeHistory(t, []Message{{"user", " \n\t"}}), 400},
		{"unicode user cap", encodeHistory(t, []Message{{"user", strings.Repeat("ñ", 4001)}}), 400},
		{"assistant cap", encodeHistory(t, []Message{{"user", "hi"}, {"assistant", strings.Repeat("a", 8001)}, {"user", "hi"}}), 400},
		{"total context cap", encodeHistory(t, []Message{{"user", strings.Repeat("u", 4000)}, {"assistant", strings.Repeat("a", 8000)}, {"user", strings.Repeat("u", 4000)}, {"assistant", "a"}, {"user", "u"}}), 400},
		{"too many messages", encodeHistory(t, tooMany), 400},
		{"unknown property", `{"messages":[{"role":"user","content":"hello","system":"override"}]}`, 400},
		{"trailing JSON", `{"messages":[{"role":"user","content":"hello"}]} {}`, 400},
		{"invalid JSON", `{`, 400},
		{"body cap", encodeHistory(t, []Message{{"user", strings.Repeat("x", maxRequestBytes)}}), 413},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			w := request(t, h, http.MethodPost, test.body, "user")
			if w.Code != test.status {
				t.Fatalf("status=%d want=%d", w.Code, test.status)
			}
		})
	}
	if calls.Load() != 0 {
		t.Fatal("invalid history reached provider")
	}
	if got := validateHistory([]Message{{"user", strings.Repeat("ñ", 4000)}}); got != "" {
		t.Fatal("character limit counted UTF-8 bytes instead of characters")
	}
}

func TestGeminiFailuresAreSanitized(t *testing.T) {
	for _, test := range []struct {
		name string
		code int
		body string
		want int
	}{
		{"invalid key", 401, `{"error":"upstream-secret-do-not-return"}`, 503},
		{"forbidden", 403, `upstream-secret-do-not-return`, 503},
		{"unknown model", 404, `upstream-secret-do-not-return`, 503},
		{"quota", 429, `upstream-secret-do-not-return`, 429},
		{"unavailable", 500, `upstream-secret-do-not-return`, 502},
		{"invalid JSON", 200, `upstream-secret-do-not-return`, 502},
		{"empty candidate", 200, `{"candidates":[]}`, 502},
		{"empty text", 200, `{"candidates":[{"content":{"parts":[{"text":"   "}]}}]}`, 502},
		{"thought only", 200, `{"candidates":[{"content":{"parts":[{"text":"private thought","thought":true}]}}]}`, 502},
		{"blocked prompt", 200, `{"promptFeedback":{"blockReason":"SAFETY"}}`, 422},
		{"blocked response", 200, `{"candidates":[{"finishReason":"SAFETY","content":{"parts":[{"text":"blocked"}]}}]}`, 422},
		{"oversized text", 200, `{"candidates":[{"content":{"parts":[{"text":"` + strings.Repeat("x", maxReplyRunes+1) + `"}]}}]}`, 502},
		{"oversized response", 200, strings.Repeat("x", maxResponseBytes+1), 502},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.code)
				_, _ = io.WriteString(w, test.body)
			}))
			defer server.Close()
			h := NewHandler(config.Config{GeminiAPIKey: "test-key"})
			h.origin = server.URL
			w := request(t, h, http.MethodPost, `{"messages":[{"role":"user","content":"Hola"}]}`, "user")
			if w.Code != test.want || strings.Contains(w.Body.String(), "upstream-secret") || strings.Contains(w.Body.String(), "private thought") || w.Header().Get("Cache-Control") != "private, no-store" {
				t.Fatalf("unexpected failure response: status=%d body=%s", w.Code, w.Body.String())
			}
			if test.code == 429 && w.Header().Get("Retry-After") == "" {
				t.Fatal("quota error lacks retry hint")
			}
		})
	}
}

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type brokenBody struct{}

func (brokenBody) Read([]byte) (int, error) { return 0, errors.New("upstream-secret-read-error") }
func (brokenBody) Close() error             { return nil }

func TestGeminiReadErrorsAndCancellation(t *testing.T) {
	h := NewHandler(config.Config{GeminiAPIKey: "test-key"})
	h.client.Transport = transportFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: brokenBody{}, Header: make(http.Header)}, nil
	})
	w := request(t, h, http.MethodPost, `{"messages":[{"role":"user","content":"Hola"}]}`, "user")
	if w.Code != 502 || strings.Contains(w.Body.String(), "upstream-secret") {
		t.Fatal("provider body read failure was hidden or leaked")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	h.client = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) { return nil, r.Context().Err() })}
	w = requestContext(t, h, ctx, http.MethodPost, `{"messages":[{"role":"user","content":"Hola"}]}`, "user")
	if w.Code != 408 {
		t.Fatalf("cancelled request status=%d", w.Code)
	}
	h.timeout = 5 * time.Millisecond
	h.client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) { <-r.Context().Done(); return nil, r.Context().Err() })
	w = request(t, h, http.MethodPost, `{"messages":[{"role":"user","content":"Hola"}]}`, "user")
	if w.Code != 504 || h.limits.global.inFlight != 0 {
		t.Fatal("timeout did not release in-flight allowance")
	}
}

func TestGeminiRedirectDoesNotForwardAPIKey(t *testing.T) {
	var leakedCalls atomic.Int32
	other := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { leakedCalls.Add(1) }))
	defer other.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL, http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	h := NewHandler(config.Config{GeminiAPIKey: "test-key"})
	h.origin = server.URL
	w := request(t, h, http.MethodPost, `{"messages":[{"role":"user","content":"Hola"}]}`, "user")
	if w.Code != http.StatusBadGateway || leakedCalls.Load() != 0 {
		t.Fatal("provider redirect forwarded an API key")
	}
}

func TestAssistantConcurrentAndRateLimitedRequestsDoNotReachProvider(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = io.WriteString(w, `{"candidates":[{"content":{"parts":[{"text":"hello"}]}}]}`)
	}))
	defer server.Close()
	h := NewHandler(config.Config{GeminiAPIKey: "test-key"})
	h.origin = server.URL
	if !h.limits.acquire("user") {
		t.Fatal("first request was rejected")
	}
	body := `{"messages":[{"role":"user","content":"Hola"}]}`
	if w := request(t, h, http.MethodPost, body, "user"); w.Code != 429 || calls.Load() != 0 {
		t.Fatal("second concurrent request reached Gemini")
	}
	h.limits.release("user")
	for i := 1; i < userMinuteLimit; i++ {
		if w := request(t, h, http.MethodPost, body, "user"); w.Code != 200 {
			t.Fatalf("admitted request %d failed", i)
		}
	}
	if w := request(t, h, http.MethodPost, body, "user"); w.Code != 429 || w.Header().Get("Retry-After") == "" || calls.Load() != userMinuteLimit-1 {
		t.Fatal("per-user quota did not block provider call")
	}
}

func TestLimiterGlobalDailyAndMemoryBounds(t *testing.T) {
	now := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	l := newLimiter()
	l.now = func() time.Time { return now }
	for i := 0; i < globalInFlight; i++ {
		if !l.acquire(fmt.Sprint(i)) {
			t.Fatal("global concurrency limit rejected an available slot")
		}
	}
	if l.acquire("extra") || l.acquire("0") {
		t.Fatal("global or per-user concurrency limit exceeded")
	}
	for i := 0; i < globalInFlight; i++ {
		l.release(fmt.Sprint(i))
	}
	l = newLimiter()
	l.now = func() time.Time { return now }
	for i := 0; i < globalDayLimit; i++ {
		if i > 0 && i%globalMinuteLimit == 0 {
			if l.acquire("blocked-in-minute") {
				t.Fatal("global minute limit exceeded")
			}
			now = now.Add(time.Minute)
		}
		id := fmt.Sprint(i)
		if !l.acquire(id) {
			t.Fatalf("request %d unexpectedly blocked", i)
		}
		l.release(id)
	}
	now = now.Add(time.Minute)
	if l.acquire("daily-overflow") {
		t.Fatal("global daily limit exceeded")
	}
	if l.retryAfter("daily-overflow") <= 60 {
		t.Fatal("daily quota returned a minute-scale retry hint")
	}
	now = now.Add(24 * time.Hour)
	if !l.acquire("new-day") {
		t.Fatal("global daily quota did not reset")
	}
	l.release("new-day")
	if len(l.users) != 1 {
		t.Fatal("expired usage entries were not cleaned")
	}
	l.users = make(map[string]*usage)
	for i := 0; i < maxTrackedUsers; i++ {
		l.users[fmt.Sprint(i)] = &usage{lastSeen: now}
	}
	if l.acquire("registry-overflow") || len(l.users) > maxTrackedUsers {
		t.Fatal("limiter registry grew without a bound")
	}
}

func TestLimiterPerUserDailyQuotaAndReset(t *testing.T) {
	now := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	l := newLimiter()
	l.now = func() time.Time { return now }
	for i := 0; i < userDayLimit; i++ {
		if i > 0 && i%userMinuteLimit == 0 {
			now = now.Add(time.Minute)
		}
		if !l.acquire("user") {
			t.Fatalf("request %d unexpectedly blocked", i)
		}
		l.release("user")
	}
	now = now.Add(time.Minute)
	if l.acquire("user") {
		t.Fatal("per-user daily quota exceeded")
	}
	if l.retryAfter("user") <= 60 {
		t.Fatal("per-user daily quota returned a minute-scale retry hint")
	}
	now = now.Add(24 * time.Hour)
	if !l.acquire("user") {
		t.Fatal("per-user daily quota did not reset")
	}
	l.release("user")
}
