package assistant

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"realtime-chat/backend/internal/auth"
	"realtime-chat/backend/internal/config"
	"realtime-chat/backend/internal/httputil"
)

const maxRequestBytes = 128 << 10

type Handler struct {
	key     string
	model   string
	origin  string
	client  *http.Client
	timeout time.Duration
	limits  *limiter
}

func NewHandler(cfg config.Config) *Handler {
	model := cfg.GeminiModel
	if model == "" {
		model = config.DefaultGeminiModel
	}
	return &Handler{
		key: strings.TrimSpace(cfg.GeminiAPIKey), model: model, origin: geminiOrigin,
		client:  &http.Client{Timeout: requestTimeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		timeout: requestTimeout,
		limits:  newLimiter(),
	}
}

func privateResponse(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("CDN-Cache-Control", "no-store")
}

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	privateResponse(w)
	if _, ok := auth.UserID(r.Context()); !ok {
		httputil.Error(w, http.StatusUnauthorized, "No autenticado")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"enabled": h.key != "", "provider": "gemini", "model": h.model})
}

func (h *Handler) Messages(w http.ResponseWriter, r *http.Request) {
	privateResponse(w)
	userID, ok := auth.UserID(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "No autenticado")
		return
	}
	if h.key == "" {
		httputil.Error(w, http.StatusServiceUnavailable, "IACHAT todavía no está disponible. Puedes seguir usando los demás chats.")
		return
	}
	var input struct {
		Messages []Message `json:"messages"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBytes))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&input)
	if err == nil {
		var extra any
		if trailing := decoder.Decode(&extra); trailing != io.EOF {
			if trailing == nil {
				err = errors.New("unexpected JSON after request")
			} else {
				err = trailing
			}
		}
	}
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			httputil.Error(w, http.StatusRequestEntityTooLarge, "El historial de la IA es demasiado grande.")
		} else {
			httputil.Error(w, http.StatusBadRequest, "JSON inválido para la conversación con IA.")
		}
		return
	}
	if message := validateHistory(input.Messages); message != "" {
		httputil.Error(w, http.StatusBadRequest, message)
		return
	}
	if !h.limits.acquire(userID) {
		w.Header().Set("Retry-After", strconv.Itoa(h.limits.retryAfter(userID)))
		httputil.Error(w, http.StatusTooManyRequests, "IACHAT alcanzó su límite de solicitudes. Intenta más tarde.")
		return
	}
	defer h.limits.release(userID)
	content, failure := h.generate(r.Context(), input.Messages)
	if failure != nil {
		if failure.retryAfter != "" {
			w.Header().Set("Retry-After", failure.retryAfter)
		}
		httputil.Error(w, failure.status, failure.message)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"content": content})
}

func validateHistory(messages []Message) string {
	if len(messages) == 0 || len(messages) > 21 || len(messages)%2 == 0 {
		return "Envía hasta 21 mensajes alternando usuario y asistente, comenzando y terminando con un mensaje de usuario."
	}
	total := 0
	for index, message := range messages {
		expected := "user"
		if index%2 == 1 {
			expected = "assistant"
		}
		if message.Role != expected || strings.TrimSpace(message.Content) == "" {
			return "El historial debe alternar mensajes de usuario y asistente con contenido."
		}
		length := utf8.RuneCountInString(message.Content)
		if (expected == "user" && length > 4000) || length > maxReplyRunes {
			return "Cada mensaje de usuario puede tener hasta 4000 caracteres y cada respuesta hasta 8000."
		}
		total += length
		if total > 16000 {
			return "El contexto de la IA puede tener hasta 16000 caracteres. Inicia una nueva conversación."
		}
	}
	return ""
}
