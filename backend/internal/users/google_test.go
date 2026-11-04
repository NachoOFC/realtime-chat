package users

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"realtime-chat/backend/internal/config"
)

type failingGoogle struct{ calls int }

func (*failingGoogle) AuthorizationURL(state, nonce, verifier string) string {
	return "https://google.test/auth?state=" + url.QueryEscape(state)
}
func (p *failingGoogle) Exchange(context.Context, string, string, string) (GoogleIdentity, error) {
	p.calls++
	return GoogleIdentity{}, errors.New("provider unavailable")
}
func TestGoogleStateAndFailureHandling(t *testing.T) {
	cfg := config.Config{JWTSecret: "test-secret", FrontendURL: "https://frontend.test", CookieSecure: true}
	provider := &failingGoogle{}
	handler := NewGoogleHandler(nil, cfg)
	handler.Provider = provider
	start := httptest.NewRecorder()
	handler.Start(start, httptest.NewRequest("GET", "https://backend.test/api/auth/google", nil))
	if start.Code != 302 {
		t.Fatal(start.Code)
	}
	cookies := start.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].SameSite != http.SameSiteLaxMode || cookies[0].MaxAge != 600 {
		t.Fatal("OAuth cookie is not protected")
	}
	location, _ := url.Parse(start.Header().Get("Location"))
	state := location.Query().Get("state")
	if len(state) < 32 {
		t.Fatal("state lacks entropy")
	}
	for _, test := range []struct{ name, state, raw, query, want string }{
		{"missing cookie", state, "", "code=test", "google_state"},
		{"wrong state", "wrong", cookies[0].Value, "code=test", "google_state"},
		{"tampered cookie", state, cookies[0].Value + "tampered", "code=test", "google_state"},
		{"user denied", state, cookies[0].Value, "error=access_denied", "google_cancelled"},
		{"missing code", state, cookies[0].Value, "", "google_failed"},
		{"provider failed", state, cookies[0].Value, "code=test", "google_failed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			before := provider.calls
			req := httptest.NewRequest("GET", "https://backend.test/api/auth/google/callback?state="+url.QueryEscape(test.state)+"&"+test.query, nil)
			if test.raw != "" {
				req.AddCookie(&http.Cookie{Name: googleFlowCookie, Value: test.raw})
			}
			out := httptest.NewRecorder()
			handler.Callback(out, req)
			if out.Code != 303 || out.Header().Get("Location") != "https://frontend.test/login?authError="+test.want {
				t.Fatal(out.Code, out.Header().Get("Location"))
			}
			if test.name != "provider failed" && provider.calls != before {
				t.Fatal("exchanged an untrusted code")
			}
			for _, cookie := range out.Result().Cookies() {
				if cookie.Name == googleFlowCookie && cookie.MaxAge != -1 {
					t.Fatal("OAuth state cookie was not cleared")
				}
			}
		})
	}
	// Expired flow cookies are rejected before contacting the provider.
	expired := googleFlow{State: state, Nonce: "nonce", Verifier: "verifier", RegisteredClaims: jwt.RegisteredClaims{Issuer: "realtime-chat/google-oauth", Audience: jwt.ClaimStrings{"google-oauth"}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute))}}
	raw, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, expired).SignedString([]byte(cfg.JWTSecret))
	req := httptest.NewRequest("GET", "https://backend.test/api/auth/google/callback?state="+state+"&code=test", nil)
	req.AddCookie(&http.Cookie{Name: googleFlowCookie, Value: raw})
	out := httptest.NewRecorder()
	handler.Callback(out, req)
	if !strings.Contains(out.Header().Get("Location"), "google_state") {
		t.Fatal("expired flow accepted")
	}
	disabled := NewGoogleHandler(nil, cfg)
	out = httptest.NewRecorder()
	disabled.Start(out, req)
	if out.Code != 503 {
		t.Fatal("disabled Google accepted login")
	}
}
func TestGoogleAuthorizationPKCE(t *testing.T) {
	provider := newGoogleProvider(config.Config{GoogleClientID: "client", GoogleClientSecret: "secret", GoogleRedirectURL: "https://backend.test/api/auth/google/callback"})
	verifier := oauth2.GenerateVerifier()
	target, err := url.Parse(provider.AuthorizationURL("state", "nonce", verifier))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(verifier))
	q := target.Query()
	if target.Host != "accounts.google.com" || q.Get("code_challenge") != base64.RawURLEncoding.EncodeToString(sum[:]) || q.Get("code_challenge_method") != "S256" || q.Get("nonce") != "nonce" || q.Get("scope") != "openid email profile" {
		t.Fatal("incorrect OAuth request")
	}
}
func TestGoogleSignedIdentityValidation(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var raw string
	verifierValue := oauth2.GenerateVerifier()
	pkceReceived := false
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/keys" {
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]string{"kty": "RSA", "kid": "test", "use": "sig", "alg": "RS256", "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())}}})
			return
		}
		_ = r.ParseForm()
		pkceReceived = r.Form.Get("code_verifier") == verifierValue
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "test-access", "token_type": "Bearer", "expires_in": 3600, "id_token": raw})
	}))
	defer server.Close()
	client := server.Client()
	keys := oidc.NewRemoteKeySet(oidc.ClientContext(context.Background(), client), server.URL+"/keys")
	provider := &googleOIDC{client: client, config: oauth2.Config{ClientID: "client", ClientSecret: "secret", RedirectURL: "https://backend.test/callback", Endpoint: oauth2.Endpoint{TokenURL: server.URL + "/token", AuthStyle: oauth2.AuthStyleInParams}}, verifier: oidc.NewVerifier("https://accounts.google.com", keys, &oidc.Config{ClientID: "client", SupportedSigningAlgs: []string{oidc.RS256}})}
	for _, test := range []struct {
		name     string
		change   func(jwt.MapClaims)
		wrongKey bool
		valid    bool
	}{
		{name: "valid", valid: true},
		{name: "wrong audience", change: func(c jwt.MapClaims) { c["aud"] = "someone-else" }},
		{name: "wrong issuer", change: func(c jwt.MapClaims) { c["iss"] = "https://attacker.test" }},
		{name: "expired", change: func(c jwt.MapClaims) { c["exp"] = time.Now().Add(-time.Hour).Unix() }},
		{name: "wrong nonce", change: func(c jwt.MapClaims) { c["nonce"] = "another-session" }},
		{name: "unverified email", change: func(c jwt.MapClaims) { c["email_verified"] = false }},
		{name: "wrong authorized party", change: func(c jwt.MapClaims) { c["azp"] = "another-client" }},
		{name: "wrong signing key", wrongKey: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			claims := jwt.MapClaims{"iss": "https://accounts.google.com", "aud": "client", "sub": "stable-google-id", "exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(), "nonce": "nonce", "email": "person@example.test", "email_verified": true, "name": "Persona", "picture": "https://example.test/avatar"}
			if test.change != nil {
				test.change(claims)
			}
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
			token.Header["kid"] = "test"
			signingKey := key
			if test.wrongKey {
				signingKey = otherKey
			}
			raw, err = token.SignedString(signingKey)
			if err != nil {
				t.Fatal(err)
			}
			identity, err := provider.Exchange(context.Background(), "code", "nonce", verifierValue)
			if test.valid {
				if err != nil || identity.Subject != "stable-google-id" || !pkceReceived {
					t.Fatal(identity, err, pkceReceived)
				}
			} else if err == nil {
				t.Fatal("untrusted identity accepted")
			}
		})
	}
}
