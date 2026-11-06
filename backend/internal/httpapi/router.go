package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"realtime-chat/backend/internal/assistant"
	"realtime-chat/backend/internal/auth"
	"realtime-chat/backend/internal/chat"
	"realtime-chat/backend/internal/config"
	"realtime-chat/backend/internal/httputil"
	"realtime-chat/backend/internal/users"
	ws "realtime-chat/backend/internal/websocket"
)

type Dependencies struct {
	Assistant      *assistant.Handler
	Google         *users.GoogleHandler
	Config         config.Config
	AuthMiddleware func(http.Handler) http.Handler
	Users          *users.Handler
	Chat           *chat.Handler
	WebSocket      *ws.Handler
}

func NewRouter(deps Dependencies) http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			rw := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(rw, r)
			slog.Info("request", "method", r.Method, "path", r.URL.Path, "status", rw.Status(), "duration", time.Since(started))
		})
	})
	router.Use(middleware.Recoverer)
	router.Use(auth.OriginGuard(deps.Config.CORSOrigin))
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{deps.Config.CORSOrigin},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	router.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		httputil.Error(w, http.StatusNotFound, "Ruta no encontrada")
	})
	router.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		httputil.Error(w, http.StatusMethodNotAllowed, "Método no permitido")
	})

	router.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	router.Get("/ws", deps.WebSocket.ServeWS)

	router.Route("/api", func(router chi.Router) {
		router.Post("/auth/guest", deps.Users.Guest)
		router.Post("/auth/register", deps.Users.Register)
		router.Post("/auth/login", deps.Users.Login)
		google := deps.Google
		if google == nil {
			google = users.NewGoogleHandler(nil, deps.Config)
		}
		router.Get("/auth/providers", google.Status)
		router.Get("/auth/google", google.Start)
		router.Get("/auth/google/callback", google.Callback)

		router.Group(func(router chi.Router) {
			router.Use(deps.AuthMiddleware)
			ai := deps.Assistant
			if ai == nil {
				ai = assistant.NewHandler(deps.Config)
			}
			router.Get("/assistant", ai.Status)
			router.Post("/assistant/messages", ai.Messages)
			router.Post("/auth/logout", deps.Users.Logout)

			router.Get("/me", deps.Users.Me)
			router.Get("/users", deps.Users.List)
			router.Post("/ws/ticket", deps.WebSocket.IssueTicket)

			router.Route("/conversations", func(router chi.Router) {
				router.Get("/", deps.Chat.ListConversations)
				router.Post("/", deps.Chat.CreateConversation)
				router.Post("/groups", deps.Chat.CreateGroup)
				router.Post("/{id}/members", deps.Chat.AddGroupMembers)
				router.Delete("/{id}/members/me", deps.Chat.LeaveGroup)
				router.Get("/{id}/messages", deps.Chat.ListMessages)
				router.Delete("/{id}", deps.Chat.DeleteConversation)
				router.Delete("/{id}/messages/{messageId}", deps.Chat.DeleteMessage)
			})
		})
	})

	return router
}
