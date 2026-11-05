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
	service   *Service
	OnChanged func(conversationID string, userIDs []string, removedUserID string)
}

func (h *Handler) notifyGroup(c Conversation) {
	if h.OnChanged == nil {
		return
	}
	ids := make([]string, 0, len(c.Members))
	for _, member := range c.Members {
		ids = append(ids, member.ID)
	}
	h.OnChanged(c.ID, ids, "")
}

func (h *Handler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.UserID(r.Context())
	if !ok {
		httputil.Error(w, 401, "No autenticado")
		return
	}
	var input CreateGroupInput
	if !httputil.ReadJSON(w, r, &input) {
		return
	}
	c, err := h.service.CreateGroup(r.Context(), actor, input.Name, input.MemberIDs)
	if err != nil {
		writeError(w, err)
		return
	}
	h.notifyGroup(c)
	httputil.WriteJSON(w, http.StatusCreated, c)
}

func (h *Handler) AddGroupMembers(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.UserID(r.Context())
	if !ok {
		httputil.Error(w, 401, "No autenticado")
		return
	}
	var input AddGroupMembersInput
	if !httputil.ReadJSON(w, r, &input) {
		return
	}
	c, err := h.service.AddGroupMembers(r.Context(), actor, chi.URLParam(r, "id"), input.MemberIDs)
	if err != nil {
		writeError(w, err)
		return
	}
	h.notifyGroup(c)
	httputil.WriteJSON(w, http.StatusOK, c)
}

func (h *Handler) LeaveGroup(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.UserID(r.Context())
	if !ok {
		httputil.Error(w, 401, "No autenticado")
		return
	}
	id := chi.URLParam(r, "id")
	ids, err := h.service.LeaveGroup(r.Context(), actor, id)
	if err != nil {
		writeError(w, err)
		return
	}
	if h.OnChanged != nil {
		h.OnChanged(id, ids, actor)
	}
	w.WriteHeader(http.StatusNoContent)
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
	case errors.Is(err, ErrClosed):
		httputil.Error(w, http.StatusGone, err.Error())
	case errors.Is(err, ErrConflict):
		httputil.Error(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrForbidden):
		httputil.Error(w, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrNotFound):
		httputil.Error(w, http.StatusNotFound, err.Error())
	default:
		httputil.Error(w, http.StatusInternalServerError, "Error interno")
	}
}
