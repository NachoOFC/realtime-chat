package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"realtime-chat/backend/internal/chat"
	"realtime-chat/backend/internal/config"
	"realtime-chat/backend/internal/httputil"
	"realtime-chat/backend/internal/users"
	ws "realtime-chat/backend/internal/websocket"
)

type Dependencies struct {
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
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
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
		router.Post("/auth/logout", deps.Users.Logout)

		router.Group(func(router chi.Router) {
			router.Use(deps.AuthMiddleware)

			router.Get("/me", deps.Users.Me)
			router.Get("/users", deps.Users.List)
			router.Post("/ws/ticket", deps.WebSocket.IssueTicket)

			router.Route("/conversations", func(router chi.Router) {
				router.Get("/", deps.Chat.ListConversations)
				router.Post("/", deps.Chat.CreateConversation)
				router.Get("/{id}/messages", deps.Chat.ListMessages)
			})
		})
	})

	return router
}
