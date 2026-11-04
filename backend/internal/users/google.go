package users

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"realtime-chat/backend/internal/auth"
	"realtime-chat/backend/internal/config"
	"realtime-chat/backend/internal/httputil"
)

const googleFlowCookie = "chat_google_flow"
const flowLifetime = 10 * time.Minute

type GoogleProvider interface {
	AuthorizationURL(state, nonce, verifier string) string
	Exchange(context.Context, string, string, string) (GoogleIdentity, error)
}
type googleOIDC struct {
	client   *http.Client
	config   oauth2.Config
	verifier *oidc.IDTokenVerifier
}

func newGoogleProvider(cfg config.Config) GoogleProvider {
	client := &http.Client{Timeout: 10 * time.Second}
	keys := oidc.NewRemoteKeySet(oidc.ClientContext(context.Background(), client), "https://www.googleapis.com/oauth2/v3/certs")
	return &googleOIDC{client: client, config: oauth2.Config{ClientID: cfg.GoogleClientID, ClientSecret: cfg.GoogleClientSecret, RedirectURL: cfg.GoogleRedirectURL,
		Scopes: []string{oidc.ScopeOpenID, "email", "profile"}, Endpoint: oauth2.Endpoint{AuthURL: "https://accounts.google.com/o/oauth2/v2/auth", TokenURL: "https://oauth2.googleapis.com/token"}},
		verifier: oidc.NewVerifier("https://accounts.google.com", keys, &oidc.Config{ClientID: cfg.GoogleClientID, SupportedSigningAlgs: []string{oidc.RS256}})}
}
func (p *googleOIDC) AuthorizationURL(state, nonce, verifier string) string {
	return p.config.AuthCodeURL(state, oauth2.SetAuthURLParam("nonce", nonce), oauth2.SetAuthURLParam("prompt", "select_account"), oauth2.S256ChallengeOption(verifier))
}
func (p *googleOIDC) Exchange(ctx context.Context, code, nonce, verifier string) (GoogleIdentity, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, p.client)
	token, err := p.config.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return GoogleIdentity{}, err
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok {
		return GoogleIdentity{}, errors.New("missing Google ID token")
	}
	id, err := p.verifier.Verify(ctx, raw)
	if err != nil {
		return GoogleIdentity{}, err
	}
	if subtle.ConstantTimeCompare([]byte(id.Nonce), []byte(nonce)) != 1 {
		return GoogleIdentity{}, errors.New("Google nonce mismatch")
	}
	var claims struct {
		Email    string `json:"email"`
		Verified bool   `json:"email_verified"`
		Name     string `json:"name"`
		Picture  string `json:"picture"`
		Party    string `json:"azp"`
	}
	if err = id.Claims(&claims); err != nil {
		return GoogleIdentity{}, err
	}
	if !claims.Verified || claims.Email == "" || id.Subject == "" || (claims.Party != "" && claims.Party != p.config.ClientID) {
		return GoogleIdentity{}, errors.New("unverified Google identity")
	}
	if avatar, err := url.Parse(claims.Picture); err != nil || avatar.Scheme != "https" {
		claims.Picture = ""
	}
	return GoogleIdentity{Subject: id.Subject, Email: claims.Email, Name: claims.Name, Picture: claims.Picture}, nil
}

type googleFlow struct {
	State    string `json:"state"`
	Nonce    string `json:"nonce"`
	Verifier string `json:"verifier"`
	jwt.RegisteredClaims
}
type GoogleHandler struct {
	service  *Service
	cfg      config.Config
	Provider GoogleProvider
}

func NewGoogleHandler(service *Service, cfg config.Config) *GoogleHandler {
	h := &GoogleHandler{service: service, cfg: cfg}
	if service != nil && cfg.GoogleEnabled() {
		h.Provider = newGoogleProvider(cfg)
	}
	return h
}
func (h *GoogleHandler) Status(w http.ResponseWriter, r *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, map[string]bool{"google": h.Provider != nil})
}
func (h *GoogleHandler) Start(w http.ResponseWriter, r *http.Request) {
	if h.Provider == nil {
		httputil.Error(w, http.StatusServiceUnavailable, "Google todavía no está configurado")
		return
	}
	now := time.Now()
	flow := googleFlow{State: oauth2.GenerateVerifier(), Nonce: oauth2.GenerateVerifier(), Verifier: oauth2.GenerateVerifier(), RegisteredClaims: jwt.RegisteredClaims{
		Issuer: "realtime-chat/google-oauth", Audience: jwt.ClaimStrings{"google-oauth"}, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(flowLifetime))}}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, flow).SignedString([]byte(h.cfg.JWTSecret))
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "No se pudo iniciar Google")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: googleFlowCookie, Value: signed, Path: "/api/auth/google", Domain: h.cfg.CookieDomain, HttpOnly: true, Secure: h.cfg.CookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: int(flowLifetime.Seconds())})
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	http.Redirect(w, r, h.Provider.AuthorizationURL(flow.State, flow.Nonce, flow.Verifier), http.StatusFound)
}
func (h *GoogleHandler) redirect(w http.ResponseWriter, r *http.Request, path, reason string) {
	target := h.cfg.FrontendURL + path
	if reason != "" {
		target += "?authError=" + url.QueryEscape(reason)
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	http.Redirect(w, r, target, http.StatusSeeOther)
}
func (h *GoogleHandler) Callback(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(googleFlowCookie)
	http.SetCookie(w, &http.Cookie{Name: googleFlowCookie, Value: "", Path: "/api/auth/google", Domain: h.cfg.CookieDomain, HttpOnly: true, Secure: h.cfg.CookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	if h.Provider == nil {
		h.redirect(w, r, "/login", "google_unavailable")
		return
	}
	if err != nil {
		h.redirect(w, r, "/login", "google_state")
		return
	}
	var flow googleFlow
	_, err = jwt.ParseWithClaims(cookie.Value, &flow, func(_ *jwt.Token) (any, error) { return []byte(h.cfg.JWTSecret), nil }, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer("realtime-chat/google-oauth"), jwt.WithAudience("google-oauth"), jwt.WithExpirationRequired())
	if err != nil || flow.State == "" || flow.Nonce == "" || flow.Verifier == "" || subtle.ConstantTimeCompare([]byte(flow.State), []byte(r.URL.Query().Get("state"))) != 1 {
		h.redirect(w, r, "/login", "google_state")
		return
	}
	if r.URL.Query().Get("error") != "" {
		h.redirect(w, r, "/login", "google_cancelled")
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		h.redirect(w, r, "/login", "google_failed")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	identity, err := h.Provider.Exchange(ctx, code, flow.Nonce, flow.Verifier)
	if err != nil {
		h.redirect(w, r, "/login", "google_failed")
		return
	}
	user, token, err := h.service.GoogleLogin(ctx, identity)
	if errors.Is(err, ErrEmailInUse) {
		h.redirect(w, r, "/login", "google_email_in_use")
		return
	}
	if err != nil || user.IsGuest {
		h.redirect(w, r, "/login", "google_failed")
		return
	}
	auth.SetSessionCookie(w, token, h.cfg.CookieDomain, h.cfg.CookieSecure, auth.SameSite(h.cfg.CookieSameSite), h.cfg.AccessTokenTTL)
	h.redirect(w, r, "/chat", "")
}
