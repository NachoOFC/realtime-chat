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
	tickets := ws.NewTicketStore(cfg.WSTicketTTL)
	websocketHandler := ws.NewHandler(hub, tickets, chatService, cfg.CORSOrigin)

	router := httpapi.NewRouter(httpapi.Dependencies{
		Config:         cfg,
		AuthMiddleware: auth.Middleware(cfg.JWTSecret),
		Users:          usersHandler,
		Chat:           chatHandler,
		WebSocket:      websocketHandler,
	})

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("servidor escuchando", "addr", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("servidor detenido", "error", err)
			stop()
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("apagado forzado", "error", err)
	}
}
