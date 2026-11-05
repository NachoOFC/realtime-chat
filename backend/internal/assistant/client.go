package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	geminiOrigin     = "https://generativelanguage.googleapis.com"
	requestTimeout   = 22 * time.Second
	maxResponseBytes = 1 << 20
	maxReplyRunes    = 8000
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type geminiPart struct {
	Text    string `json:"text"`
	Thought bool   `json:"thought,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiRequest struct {
	Contents          []geminiContent `json:"contents"`
	SystemInstruction geminiContent   `json:"systemInstruction"`
	GenerationConfig  struct {
		MaxOutputTokens int `json:"maxOutputTokens"`
		CandidateCount  int `json:"candidateCount"`
	} `json:"generationConfig"`
}

type geminiResponse struct {
	Candidates []struct {
		Content      geminiContent `json:"content"`
		FinishReason string        `json:"finishReason"`
	} `json:"candidates"`
	PromptFeedback struct {
		BlockReason string `json:"blockReason"`
	} `json:"promptFeedback"`
}

type responseError struct {
	status     int
	message    string
	retryAfter string
}

func (e *responseError) Error() string { return e.message }

func (h *Handler) generate(ctx context.Context, messages []Message) (string, *responseError) {
	ctx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()
	payload := geminiRequest{
		SystemInstruction: geminiContent{Parts: []geminiPart{{Text: "Eres IACHAT, el asistente de Chatxor. Responde de forma clara y breve en español latinoamericano, salvo que la persona pida otro idioma. Puedes ayudar a conversar, explicar ideas y resolver preguntas. Reconoce cuando no sabes algo. No tienes acceso a otros chats, usuarios, mensajes privados, cuentas ni bases de datos: solo recibes esta conversación. No afirmes haber realizado acciones fuera de este chat."}}},
	}
	payload.GenerationConfig.MaxOutputTokens = 1024
	payload.GenerationConfig.CandidateCount = 1
	for _, message := range messages {
		role := message.Role
		if role == "assistant" {
			role = "model"
		}
		payload.Contents = append(payload.Contents, geminiContent{Role: role, Parts: []geminiPart{{Text: message.Content}}})
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", unavailableResponse()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.origin+"/v1beta/models/"+url.PathEscape(h.model)+":generateContent", bytes.NewReader(body))
	if err != nil {
		return "", unavailableResponse()
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-goog-api-key", h.key)
	response, err := h.client.Do(req)
	if err != nil {
		return "", requestFailure(ctx)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		switch response.StatusCode {
		case http.StatusTooManyRequests:
			return "", &responseError{http.StatusTooManyRequests, "Se alcanzó el límite de uso de la IA. Intenta más tarde.", "60"}
		case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound:
			return "", unavailableResponse()
		default:
			return "", &responseError{status: http.StatusBadGateway, message: "La IA no pudo responder. Intenta de nuevo."}
		}
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return "", requestFailure(ctx)
	}
	if len(raw) > maxResponseBytes {
		return "", invalidResponse()
	}
	var result geminiResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", invalidResponse()
	}
	if result.PromptFeedback.BlockReason != "" && result.PromptFeedback.BlockReason != "BLOCK_REASON_UNSPECIFIED" {
		return "", blockedResponse()
	}
	if len(result.Candidates) == 0 {
		return "", invalidResponse()
	}
	candidate := result.Candidates[0]
	if candidate.FinishReason != "" && candidate.FinishReason != "STOP" && candidate.FinishReason != "MAX_TOKENS" {
		return "", blockedResponse()
	}
	var reply strings.Builder
	for _, part := range candidate.Content.Parts {
		if !part.Thought {
			reply.WriteString(part.Text)
		}
	}
	content := strings.TrimSpace(reply.String())
	if content == "" || utf8.RuneCountInString(content) > maxReplyRunes {
		return "", invalidResponse()
	}
	return content, nil
}

func unavailableResponse() *responseError {
	return &responseError{status: http.StatusServiceUnavailable, message: "IACHAT no está disponible en este momento. Intenta más tarde."}
}

func invalidResponse() *responseError {
	return &responseError{status: http.StatusBadGateway, message: "La IA devolvió una respuesta no válida. Intenta de nuevo."}
}

func blockedResponse() *responseError {
	return &responseError{status: http.StatusUnprocessableEntity, message: "La IA no pudo responder a este mensaje. Intenta reformularlo."}
}

func requestFailure(ctx context.Context) *responseError {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return &responseError{status: http.StatusGatewayTimeout, message: "La IA tardó demasiado en responder. Intenta de nuevo."}
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return &responseError{status: http.StatusRequestTimeout, message: "La solicitud a la IA fue cancelada."}
	}
	return &responseError{status: http.StatusBadGateway, message: "No se pudo conectar con la IA. Intenta de nuevo."}
}
