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
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
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
	for _, name := range []string{"0001_init.up.sql", "0002_guest_users.up.sql", "0003_google_and_temporary_guests.up.sql", "0004_group_conversations.up.sql", "0005_chat_deletion.up.sql"} {
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
	t.Run("groups isolate members and survive guest cleanup", func(t *testing.T) {
		second, _, err := service.GoogleLogin(ctx, GoogleIdentity{Subject: "group-second", Email: "group-second@example.test", Name: "Segunda"})
		if err != nil {
			t.Fatal(err)
		}
		third, _, err := service.GoogleLogin(ctx, GoogleIdentity{Subject: "group-third", Email: "group-third@example.test", Name: "Tercera"})
		if err != nil {
			t.Fatal(err)
		}
		temporary, _, err := service.GuestLogin(ctx, "GroupGuest")
		if err != nil {
			t.Fatal(err)
		}
		group, err := chats.CreateGroup(ctx, account.ID, "Amigos", []string{second.ID, temporary.ID})
		if err != nil {
			t.Fatal(err)
		}
		if group.Kind != "group" || group.Name != "Amigos" || len(group.Members) != 3 {
			t.Fatal("incorrect group metadata", group)
		}
		for _, member := range group.Members {
			if member.AuthProvider == "" {
				t.Fatal("missing member provider")
			}
		}
		if _, err = chats.ListMessages(ctx, third.ID, group.ID, "", 30); !errors.Is(err, chat.ErrForbidden) {
			t.Fatal("outsider can read", err)
		}
		if _, err = chats.SendMessage(ctx, third.ID, group.ID, uuid.NewString(), "No"); !errors.Is(err, chat.ErrForbidden) {
			t.Fatal("outsider can send", err)
		}
		if _, err = chats.AddGroupMembers(ctx, third.ID, group.ID, []string{account.ID}); !errors.Is(err, chat.ErrForbidden) {
			t.Fatal("outsider can invite", err)
		}
		if _, err = chats.LeaveGroup(ctx, third.ID, group.ID); !errors.Is(err, chat.ErrForbidden) {
			t.Fatal("outsider can change membership", err)
		}
		stored, err := chats.SendMessage(ctx, second.ID, group.ID, uuid.NewString(), "Hola grupo")
		if err != nil || stored.SenderName != second.Username {
			t.Fatal("missing group message author", err)
		}
		page, err := chats.ListMessages(ctx, account.ID, group.ID, "", 30)
		if err != nil || len(page.Messages) != 1 || page.Messages[0].SenderName != second.Username {
			t.Fatal("group history", err)
		}
		updated, err := chats.AddGroupMembers(ctx, account.ID, group.ID, []string{third.ID})
		if err != nil || len(updated.Members) != 4 {
			t.Fatal("could not invite", err)
		}
		if _, err = chats.ListMessages(ctx, third.ID, group.ID, "", 30); err != nil {
			t.Fatal("invited member cannot read", err)
		}
		if _, err = chats.SendMessage(ctx, temporary.ID, group.ID, uuid.NewString(), "Temporal"); err != nil {
			t.Fatal(err)
		}
		if _, err = service.DeleteGuest(ctx, temporary.ID); err != nil {
			t.Fatal(err)
		}
		page, err = chats.ListMessages(ctx, account.ID, group.ID, "", 30)
		if err != nil || len(page.Messages) != 1 || page.Messages[0].ID != stored.ID {
			t.Fatal("guest cleanup removed another group member's message", err)
		}
		list, err := chats.ListConversations(ctx, account.ID)
		if err != nil {
			t.Fatal(err)
		}
		count := 0
		for _, c := range list {
			if c.ID == group.ID {
				count++
				if len(c.Members) != 3 {
					t.Fatal("deleted guest still listed")
				}
			}
		}
		if count != 1 {
			t.Fatal("duplicate or missing group rows", count)
		}
		if _, err = chats.LeaveGroup(ctx, second.ID, group.ID); err != nil {
			t.Fatal(err)
		}
		if _, err = chats.ListMessages(ctx, second.ID, group.ID, "", 30); !errors.Is(err, chat.ErrForbidden) {
			t.Fatal("former member can read", err)
		}
		if _, err = chats.SendMessage(ctx, second.ID, group.ID, uuid.NewString(), "No"); !errors.Is(err, chat.ErrForbidden) {
			t.Fatal("former member can send", err)
		}
		page, err = chats.ListMessages(ctx, account.ID, group.ID, "", 30)
		if err != nil || page.Messages[0].SenderName != second.Username {
			t.Fatal("former member's author name lost", err)
		}
		emptyGuest, _, err := service.GuestLogin(ctx, "EmptyGroupGuest")
		if err != nil {
			t.Fatal(err)
		}
		emptyGroup, err := chats.CreateGroup(ctx, account.ID, "Vacío", []string{emptyGuest.ID})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = service.DeleteGuest(ctx, emptyGuest.ID); err != nil {
			t.Fatal(err)
		}
		list, err = chats.ListConversations(ctx, account.ID)
		if err != nil || !hasConversation(list, emptyGroup.ID) {
			t.Fatal("empty group was deleted with a guest", err)
		}
		if _, err = chats.SendMessage(ctx, account.ID, emptyGroup.ID, uuid.NewString(), "Sigo aquí"); err != nil {
			t.Fatal("single remaining group member cannot write", err)
		}
		if _, err = chats.LeaveGroup(ctx, account.ID, emptyGroup.ID); err != nil {
			t.Fatal(err)
		}
		var exists bool
		if err = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM conversations WHERE id=$1)`, emptyGroup.ID).Scan(&exists); err != nil || exists {
			t.Fatal("empty group remains after its last member leaves", err)
		}
		if _, err = chats.CreateGroup(ctx, account.ID, "", []string{third.ID}); !errors.Is(err, chat.ErrValidation) {
			t.Fatal("accepted empty group name", err)
		}
		if _, err = chats.CreateGroup(ctx, account.ID, "Duplicados", []string{third.ID, third.ID}); !errors.Is(err, chat.ErrValidation) {
			t.Fatal("accepted duplicate members", err)
		}
	})
	t.Run("message deletion and private chat deletion", func(t *testing.T) {
		other, _, err := service.GoogleLogin(ctx, GoogleIdentity{Subject: "deletion-other", Email: "deletion-other@example.test", Name: "Otra"})
		if err != nil {
			t.Fatal(err)
		}
		outsider, _, err := service.GoogleLogin(ctx, GoogleIdentity{Subject: "deletion-outsider", Email: "deletion-outsider@example.test", Name: "Fuera"})
		if err != nil {
			t.Fatal(err)
		}
		conversation, err := chats.CreateDirect(ctx, account.ID, other.ID)
		if err != nil {
			t.Fatal(err)
		}
		own, err := chats.SendMessage(ctx, account.ID, conversation.ID, uuid.NewString(), "Eliminar para todos")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = chats.SendMessage(ctx, other.ID, conversation.ID, uuid.NewString(), "Conservar para la otra persona"); err != nil {
			t.Fatal(err)
		}
		if _, err = chats.DeleteMessage(ctx, other.ID, conversation.ID, own.ID); !errors.Is(err, chat.ErrForbidden) {
			t.Fatal("participant deleted another person's message", err)
		}
		if _, err = chats.DeleteMessage(ctx, outsider.ID, conversation.ID, own.ID); !errors.Is(err, chat.ErrForbidden) {
			t.Fatal("outsider deleted a message", err)
		}
		if err = chats.DeleteConversation(ctx, outsider.ID, conversation.ID); !errors.Is(err, chat.ErrForbidden) {
			t.Fatal("outsider cleared a chat", err)
		}
		if err = chats.MarkRead(ctx, other.ID, conversation.ID, own.ID); err != nil {
			t.Fatal(err)
		}
		participants, err := chats.DeleteMessage(ctx, account.ID, conversation.ID, own.ID)
		if err != nil || len(participants) != 2 {
			t.Fatal("deletion recipients", err)
		}
		page, err := chats.ListMessages(ctx, other.ID, conversation.ID, "", 30)
		if err != nil || len(page.Messages) != 1 || page.Messages[0].ID == own.ID {
			t.Fatal("deleted message survived", err)
		}
		if err = chats.DeleteConversation(ctx, account.ID, conversation.ID); err != nil {
			t.Fatal(err)
		}
		list, err := chats.ListConversations(ctx, account.ID)
		if err != nil || hasConversation(list, conversation.ID) {
			t.Fatal("cleared chat still listed", err)
		}
		page, err = chats.ListMessages(ctx, account.ID, conversation.ID, "", 30)
		if err != nil || len(page.Messages) != 0 {
			t.Fatal("cleared history visible to actor", err)
		}
		page, err = chats.ListMessages(ctx, other.ID, conversation.ID, "", 30)
		if err != nil || len(page.Messages) != 1 {
			t.Fatal("another person's history was cleared", err)
		}
		fresh, err := chats.SendMessage(ctx, other.ID, conversation.ID, uuid.NewString(), "Nuevo mensaje")
		if err != nil {
			t.Fatal(err)
		}
		list, err = chats.ListConversations(ctx, account.ID)
		if err != nil || !hasConversation(list, conversation.ID) {
			t.Fatal("new incoming message did not restore chat", err)
		}
		page, err = chats.ListMessages(ctx, account.ID, conversation.ID, "", 30)
		if err != nil || len(page.Messages) != 1 || page.Messages[0].ID != fresh.ID {
			t.Fatal("old cleared history resurfaced", err)
		}
		page, err = chats.ListMessages(ctx, other.ID, conversation.ID, "", 30)
		if err != nil || len(page.Messages) != 2 {
			t.Fatal("recipient lost history", err)
		}
		if err = chats.DeleteConversation(ctx, account.ID, conversation.ID); err != nil {
			t.Fatal(err)
		}
		if _, err = chats.CreateDirect(ctx, account.ID, other.ID); err != nil {
			t.Fatal(err)
		}
		list, err = chats.ListConversations(ctx, account.ID)
		if err != nil || !hasConversation(list, conversation.ID) {
			t.Fatal("could not reopen chat", err)
		}
		page, err = chats.ListMessages(ctx, account.ID, conversation.ID, "", 30)
		if err != nil || len(page.Messages) != 0 {
			t.Fatal("reopening restored cleared messages", err)
		}
	})
	t.Log("Google recovery, groups, private deletions and authorization verified")
}

func hasConversation(conversations []chat.Conversation, id string) bool {
	for _, conversation := range conversations {
		if conversation.ID == id {
			return true
		}
	}
	return false
}
