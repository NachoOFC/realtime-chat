package chat

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"realtime-chat/backend/internal/auth"
	"realtime-chat/backend/internal/httputil"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListConversations(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "No autenticado")
		return
	}

	conversations, err := h.service.ListConversations(r.Context(), userID)
	if err != nil {
		writeError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, conversations)
}

func (h *Handler) CreateConversation(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "No autenticado")
		return
	}

	var input CreateConversationInput
	if !httputil.ReadJSON(w, r, &input) {
		return
	}

	conversation, err := h.service.CreateDirect(r.Context(), userID, input.PeerID)
	if err != nil {
		writeError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, conversation)
}

func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "No autenticado")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	page, err := h.service.ListMessages(
		r.Context(),
		userID,
		chi.URLParam(r, "id"),
		r.URL.Query().Get("before"),
		limit,
	)
	if err != nil {
		writeError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, page)
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		httputil.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrForbidden):
		httputil.Error(w, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrNotImplemented):
		httputil.Error(w, http.StatusNotImplemented, "Repositorio de chat pendiente")
	default:
		httputil.Error(w, http.StatusInternalServerError, "Error interno")
	}
}
