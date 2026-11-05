package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"realtime-chat/backend/internal/assistant"
	"realtime-chat/backend/internal/auth"
	"realtime-chat/backend/internal/chat"
	"realtime-chat/backend/internal/config"
	"realtime-chat/backend/internal/db"
	"realtime-chat/backend/internal/httpapi"
	"realtime-chat/backend/internal/users"
	ws "realtime-chat/backend/internal/websocket"
)

func main() {
	_ = godotenv.Load("../.env", ".env")

	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		slog.Error("configuración inválida", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("no se pudo conectar a postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	usersRepository := users.NewPostgresRepository(pool)
	usersService := users.NewService(usersRepository, cfg.JWTSecret, cfg.AccessTokenTTL)
	usersHandler := users.NewHandler(
		usersService,
		cfg.CookieDomain,
		cfg.CookieSecure,
		auth.SameSite(cfg.CookieSameSite),
		cfg.AccessTokenTTL,
	)

	chatRepository := chat.NewPostgresRepository(pool)
	chatService := chat.NewService(chatRepository)
	chatHandler := chat.NewHandler(chatService)

	hub := ws.NewHub()
	chatHandler.OnChanged = func(id string, userIDs []string, removedUserID string) {
		if removedUserID != "" {
			hub.RemoveUserFromConversation(removedUserID, id)
		}
		for _, userID := range userIDs {
			hub.SendToUser(userID, ws.OutboundEvent{Type: ws.EventConversationUpdated, ConversationID: id})
		}
	}
	tickets := ws.NewTicketStore(cfg.WSTicketTTL)
	websocketHandler := ws.NewHandler(hub, tickets, chatService, cfg.CORSOrigin)
	websocketHandler.ActiveUser = usersService.IsActive
	usersHandler.OnGuestDeleted = func(deleted users.DeletedGuest) {
		hub.Broadcast(ws.OutboundEvent{Type: ws.EventUserRemoved, UserID: deleted.UserID})
		hub.DisconnectUser(deleted.UserID)
	}
	go func() {
		cleanup := func() {
			if err := usersService.ExpireGuests(ctx, usersHandler.OnGuestDeleted); err != nil && ctx.Err() == nil {
				slog.Error("No se pudieron eliminar invitados vencidos", "error", err)
			}
		}
		cleanup()
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cleanup()
			}
		}
	}()

	router := httpapi.NewRouter(httpapi.Dependencies{
		Assistant:      assistant.NewHandler(cfg),
		Config:         cfg,
		AuthMiddleware: auth.Middleware(cfg.JWTSecret, usersService.IsActive),
		Google:         users.NewGoogleHandler(usersService, cfg),
		Users:          usersHandler,
		Chat:           chatHandler,
		WebSocket:      websocketHandler,
	})

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		TLSConfig:         cfg.TLSConfig(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		if cfg.TLSCertFile != "" {
			slog.Info("servidor HTTPS escuchando", "addr", cfg.HTTPAddr)
			serverErrors <- server.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
		} else {
			slog.Info("servidor HTTP escuchando", "addr", cfg.HTTPAddr)
			serverErrors <- server.ListenAndServe()
		}
	}()
	var serveError error
	select {
	case <-ctx.Done():
	case serveError = <-serverErrors:
		if !errors.Is(serveError, http.ErrServerClosed) {
			slog.Error("servidor detenido", "error", serveError)
		}
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("apagado forzado", "error", err)
	}

	if serveError != nil && !errors.Is(serveError, http.ErrServerClosed) {
		os.Exit(1)
	}
}
