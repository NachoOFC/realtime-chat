package users

import (
	"errors"
	"net/http"
	"time"

	"realtime-chat/backend/internal/auth"
	"realtime-chat/backend/internal/httputil"
)

type Handler struct {
	service        *Service
	cookieDomain   string
	cookieSecure   bool
	cookieSameSite http.SameSite
	tokenTTL       time.Duration
}

func NewHandler(service *Service, cookieDomain string, cookieSecure bool, cookieSameSite http.SameSite, tokenTTL time.Duration) *Handler {
	return &Handler{
		service:        service,
		cookieDomain:   cookieDomain,
		cookieSecure:   cookieSecure,
		cookieSameSite: cookieSameSite,
		tokenTTL:       tokenTTL,
	}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var input RegisterInput
	if !httputil.ReadJSON(w, r, &input) {
		return
	}

	user, token, err := h.service.Register(r.Context(), input)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	auth.SetSessionCookie(w, token, h.cookieDomain, h.cookieSecure, h.cookieSameSite, h.tokenTTL)
	httputil.WriteJSON(w, http.StatusCreated, user)
}

func (h *Handler) Guest(w http.ResponseWriter, r *http.Request) {
	var input GuestInput
	if !httputil.ReadJSON(w, r, &input) {
		return
	}

	user, token, err := h.service.GuestLogin(r.Context(), input.Username)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	auth.SetSessionCookie(w, token, h.cookieDomain, h.cookieSecure, h.cookieSameSite, h.tokenTTL)
	httputil.WriteJSON(w, http.StatusCreated, user)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var input LoginInput
	if !httputil.ReadJSON(w, r, &input) {
		return
	}

	user, token, err := h.service.Login(r.Context(), input)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	auth.SetSessionCookie(w, token, h.cookieDomain, h.cookieSecure, h.cookieSameSite, h.tokenTTL)
	httputil.WriteJSON(w, http.StatusOK, user)
}

func (h *Handler) Logout(w http.ResponseWriter, _ *http.Request) {
	auth.ClearSessionCookie(w, h.cookieDomain, h.cookieSecure, h.cookieSameSite)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "No autenticado")
		return
	}

	user, err := h.service.FindByID(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, user)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserID(r.Context())

	list, err := h.service.List(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, list)
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		httputil.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrInvalidCredentials):
		httputil.Error(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, ErrUsernameTaken):
		httputil.Error(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrNotFound):
		httputil.Error(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrNotImplemented):
		httputil.Error(w, http.StatusNotImplemented, "Repositorio de usuarios pendiente")
	default:
		httputil.Error(w, http.StatusInternalServerError, "Error interno")
	}
}
