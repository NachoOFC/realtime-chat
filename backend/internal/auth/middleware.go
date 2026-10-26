package auth

import (
	"context"
	"net/http"

	"realtime-chat/backend/internal/httputil"
)

type contextKey string

const userIDKey contextKey = "userID"

func Middleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(CookieName)
			if err != nil {
				httputil.Error(w, http.StatusUnauthorized, "No autenticado")
				return
			}

			userID, err := ParseToken(secret, cookie.Value)
			if err != nil {
				httputil.Error(w, http.StatusUnauthorized, "Sesión inválida")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDKey).(string)
	return userID, ok && userID != ""
}
