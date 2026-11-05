package users

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"realtime-chat/backend/internal/auth"
	"realtime-chat/backend/internal/chat"
)

func TestGoogleAccountsAndGuestCleanup(t *testing.T) {
	if os.Getenv("CHAT_INTEGRATION") != "1" {
		t.Skip("set CHAT_INTEGRATION=1 to test PostgreSQL in an isolated schema")
	}
	values, _ := godotenv.Read("../../../.env")
	dsn := os.Getenv("CHAT_TEST_DATABASE_URL")
	if dsn == "" {
		dsn = values["DATABASE_URL"]
	}
	if dsn == "" {
		t.Fatal("database configuration required")
	}
	if direct, err := url.Parse(dsn); err == nil && (direct.Scheme == "postgres" || direct.Scheme == "postgresql") {
		direct.Host = strings.Replace(direct.Host, "-pooler.", ".", 1)
		dsn = direct.String()
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal("invalid database configuration")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	admin, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal("could not initialize database")
	}
	defer admin.Close()
	schema := "auth_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		t.Fatal("could not create isolated schema", err)
	}
	defer func() {
		c, stop := context.WithTimeout(context.Background(), 15*time.Second)
		defer stop()
		if _, err := admin.Exec(c, "DROP SCHEMA "+quoted+" CASCADE"); err != nil {
			t.Error("schema cleanup", err)
		}
	}()
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, "SET search_path TO "+quoted+",public")
		return err
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var selected string
	if err = pool.QueryRow(ctx, "SELECT current_schema()").Scan(&selected); err != nil || selected != schema {
		t.Fatal("refusing to use an unisolated schema")
	}
	for _, name := range []string{"0001_init.up.sql", "0002_guest_users.up.sql", "0003_google_and_temporary_guests.up.sql"} {
		data, err := os.ReadFile(filepath.Join("../../migrations", name))
		if err != nil {
			t.Fatal(err)
		}
		if _, err = pool.Exec(ctx, strings.Replace(string(data), "CREATE EXTENSION IF NOT EXISTS pgcrypto;", "", 1)); err != nil {
			t.Fatal(err)
		}
	}
	repository := NewPostgresRepository(pool)
	service := NewService(repository, "integration-secret", time.Hour)
	guest, guestToken, err := service.GuestLogin(ctx, "Persona")
	if err != nil {
		t.Fatal(err)
	}
	if guest.AuthProvider != "guest" || !guest.IsGuest {
		t.Fatal("guest badge incorrect")
	}
	account, _, err := service.GoogleLogin(ctx, GoogleIdentity{Subject: "subject-one", Email: "google@example.test", Name: "Persona"})
	if err != nil {
		t.Fatal(err)
	}
	if account.IsGuest || account.AuthProvider != "google" || account.ID == guest.ID || account.Username == guest.Username {
		t.Fatal("Google linked to an existing guest")
	}
	recovered, _, err := service.GoogleLogin(ctx, GoogleIdentity{Subject: "subject-one", Email: "new-address@example.test", Name: "Changed"})
	if err != nil || recovered.ID != account.ID {
		t.Fatal("Google subject did not preserve identity", err)
	}
	_, _, err = service.Register(ctx, RegisterInput{Email: "local@example.test", Username: "Local", Password: "password-for-test"})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = service.GoogleLogin(ctx, GoogleIdentity{Subject: "subject-other", Email: "local@example.test", Name: "Otro"})
	if !errors.Is(err, ErrEmailInUse) {
		t.Fatal("Google linked a local account by email", err)
	}
	chats := chat.NewService(chat.NewPostgresRepository(pool))
	conversation, err := chats.CreateDirect(ctx, account.ID, guest.ID)
	if err != nil {
		t.Fatal(err)
	}
	permanent, err := chats.SendMessage(ctx, account.ID, conversation.ID, uuid.NewString(), "Permanente")
	if err != nil {
		t.Fatal(err)
	}
	temporary, err := chats.SendMessage(ctx, guest.ID, conversation.ID, uuid.NewString(), "Temporal")
	if err != nil {
		t.Fatal(err)
	}
	if err = chats.MarkRead(ctx, account.ID, conversation.ID, temporary.ID); err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(service, "", true, http.SameSiteLaxMode, time.Hour)
	removed := false
	handler.OnGuestDeleted = func(result DeletedGuest) {
		removed = result.UserID == guest.ID && len(result.ParticipantIDs) == 1 && result.ParticipantIDs[0] == account.ID
	}
	logout := auth.Middleware("integration-secret", service.IsActive)(http.HandlerFunc(handler.Logout))
	req := httptest.NewRequest("POST", "https://backend.test/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: guestToken})
	out := httptest.NewRecorder()
	logout.ServeHTTP(out, req)
	if out.Code != 204 || !removed {
		t.Fatal("guest logout did not delete the profile", out.Code, out.Body.String())
	}
	active, err := service.IsActive(ctx, guest.ID)
	if err != nil || active {
		t.Fatal("deleted guest remains active", err)
	}
	guarded := auth.Middleware("integration-secret", service.IsActive)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	out = httptest.NewRecorder()
	guarded.ServeHTTP(out, req)
	if out.Code != 401 {
		t.Fatal("old guest token can authenticate")
	}
	page, err := chats.ListMessages(ctx, account.ID, conversation.ID, "", 30)
	if err != nil || len(page.Messages) != 1 || page.Messages[0].ID != permanent.ID {
		t.Fatal("cleanup deleted another participant's message", page, err)
	}
	list, err := chats.ListConversations(ctx, account.ID)
	if err != nil || len(list) != 1 || list[0].Peer.ID != "" || list[0].UnreadCount != 0 {
		t.Fatal("closed conversation not accessible", list, err)
	}
	if _, err = chats.SendMessage(ctx, account.ID, conversation.ID, uuid.NewString(), "Closed"); !errors.Is(err, chat.ErrClosed) {
		t.Fatal("closed conversation accepted a message", err)
	}
	if _, err = repository.DeleteGuest(ctx, account.ID, false); !errors.Is(err, ErrNotFound) {
		t.Fatal("cleanup can delete a Google account", err)
	}
	for _, scenario := range []struct {
		name           string
		guestMessage   bool
		legacyDeletion bool
	}{
		{name: "empty conversation"},
		{name: "only guest messages", guestMessage: true},
		{name: "legacy empty conversation", legacyDeletion: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			temporaryGuest, _, err := service.GuestLogin(ctx, "Guest_"+uuid.NewString()[:8])
			if err != nil {
				t.Fatal(err)
			}
			temporaryChat, err := chats.CreateDirect(ctx, account.ID, temporaryGuest.ID)
			if err != nil {
				t.Fatal(err)
			}
			list, err := chats.ListConversations(ctx, account.ID)
			if err != nil || !hasConversation(list, temporaryChat.ID) {
				t.Fatal("an open conversation without messages must stay visible", err)
			}
			if _, err = chats.ListMessages(ctx, account.ID, temporaryChat.ID, "", 30); err != nil {
				t.Fatal("an open conversation without messages must stay accessible", err)
			}
			if scenario.guestMessage {
				if _, err = chats.SendMessage(ctx, temporaryGuest.ID, temporaryChat.ID, uuid.NewString(), "Temporal"); err != nil {
					t.Fatal(err)
				}
			}
			if scenario.legacyDeletion {
				// Simulate empty conversations left behind by the previous cleanup.
				_, err = pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, temporaryGuest.ID)
			} else {
				_, err = repository.DeleteGuest(ctx, temporaryGuest.ID, false)
			}
			if err != nil {
				t.Fatal(err)
			}
			list, err = chats.ListConversations(ctx, account.ID)
			if err != nil || hasConversation(list, temporaryChat.ID) {
				t.Fatal("a closed conversation without messages must disappear", err)
			}
			if !hasConversation(list, conversation.ID) {
				t.Fatal("cleanup hid the conversation with the permanent participant's messages")
			}
			_, err = chats.ListMessages(ctx, account.ID, temporaryChat.ID, "", 30)
			if scenario.legacyDeletion {
				if !errors.Is(err, chat.ErrNotFound) {
					t.Fatal("a legacy empty conversation remains accessible", err)
				}
			} else {
				var exists bool
				if queryErr := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM conversations WHERE id=$1)`, temporaryChat.ID).Scan(&exists); queryErr != nil || exists {
					t.Fatal("cleanup retained an empty conversation", queryErr)
				}
				if !errors.Is(err, chat.ErrForbidden) {
					t.Fatal("a deleted conversation remains accessible", err)
				}
			}
		})
	}
	replacement, _, err := service.GuestLogin(ctx, "Persona")
	if err != nil || replacement.ID == guest.ID {
		t.Fatal("deleted nickname cannot be reused safely", err)
	}
	expiryChat, err := chats.CreateDirect(ctx, account.ID, replacement.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, "UPDATE users SET guest_expires_at=now()-interval '1 minute' WHERE id=$1", replacement.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = chats.SendMessage(ctx, replacement.ID, expiryChat.ID, uuid.NewString(), "Expired"); !errors.Is(err, chat.ErrForbidden) {
		t.Fatal("expired guest can send before cleanup", err)
	}
	if err = service.ExpireGuests(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if active, err = service.IsActive(ctx, replacement.ID); err != nil || active {
		t.Fatal("expired guest remains active", err)
	}
	directory, err := service.List(ctx, account.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, user := range directory {
		if user.ID == guest.ID || user.ID == replacement.ID {
			t.Fatal("directory contains deleted guests")
		}
	}
	// Guest cookies have no Max-Age/Expires and Google cookies remain persistent.
	guestResponse := httptest.NewRecorder()
	handler.Guest(guestResponse, httptest.NewRequest("POST", "https://backend.test/api/auth/guest", strings.NewReader(`{"username":"CookieGuest"}`)))
	if guestResponse.Code != 201 {
		t.Fatal(guestResponse.Body.String())
	}
	for _, cookie := range guestResponse.Result().Cookies() {
		if cookie.Name == auth.CookieName && (cookie.MaxAge != 0 || !cookie.Expires.IsZero()) {
			t.Fatal("guest cookie persists on disk")
		}
	}
	t.Log("Google identity recovery, guest expiration, deletion, closed chat history and cookie policy verified")
}

func hasConversation(conversations []chat.Conversation, id string) bool {
	for _, conversation := range conversations {
		if conversation.ID == id {
			return true
		}
	}
	return false
}
