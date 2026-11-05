package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"realtime-chat/backend/internal/auth"
	"realtime-chat/backend/internal/chat"
	"realtime-chat/backend/internal/config"
	"realtime-chat/backend/internal/httpapi"
	"realtime-chat/backend/internal/users"
	ws "realtime-chat/backend/internal/websocket"
)

// Opt in with CHAT_INTEGRATION=1. A temporary schema isolates all test records;
// the configured application's public tables are never changed.
func TestChatOverTLS(t *testing.T) {
	if os.Getenv("CHAT_INTEGRATION") != "1" {
		t.Skip("set CHAT_INTEGRATION=1 to test with PostgreSQL in an isolated schema")
	}
	values, _ := godotenv.Read("../../../.env")
	databaseURL := os.Getenv("CHAT_TEST_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = values["DATABASE_URL"]
	}
	if databaseURL == "" {
		t.Fatal("configure CHAT_TEST_DATABASE_URL or DATABASE_URL in the root .env")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	// Reparse the direct URL so both DNS and TLS SNI target the direct endpoint.
	// Keeping a pooler SNI would route fresh connections through PgBouncer.
	if direct, err := url.Parse(databaseURL); err == nil && (direct.Scheme == "postgres" || direct.Scheme == "postgresql") {
		direct.Host = strings.Replace(direct.Host, "-pooler.", ".", 1)
		databaseURL = direct.String()
	}
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal("invalid database connection configuration")
	}
	// Use Neon's direct endpoint so session settings are preserved during schema isolation.
	cfg.ConnConfig.Host = strings.Replace(cfg.ConnConfig.Host, "-pooler.", ".", 1)
	admin, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal("could not initialize database connection")
	}
	defer admin.Close()
	schema := "chat_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		t.Fatal("could not create isolated schema:", err)
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 15*time.Second)
		defer stop()
		if _, err := admin.Exec(cleanup, "DROP SCHEMA "+quoted+" CASCADE"); err != nil {
			t.Error("could not remove temporary schema:", err)
		}
	}()
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, "SET search_path TO "+quoted+",public")
		return err
	}
	for _, fallback := range cfg.ConnConfig.Fallbacks {
		fallback.Host = strings.Replace(fallback.Host, "-pooler.", ".", 1)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var currentSchema string
	if err := pool.QueryRow(ctx, "SELECT current_schema()").Scan(&currentSchema); err != nil || currentSchema != schema {
		t.Fatal("database did not select the isolated schema; refusing to run migrations", currentSchema, err)
	}
	for _, name := range []string{"0001_init.up.sql", "0002_guest_users.up.sql", "0003_google_and_temporary_guests.up.sql", "0004_group_conversations.up.sql"} {
		migration, err := os.ReadFile(filepath.Join("../../migrations", name))
		if err != nil {
			t.Fatal(err)
		}
		sql := strings.Replace(string(migration), "CREATE EXTENSION IF NOT EXISTS pgcrypto;", "", 1)
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatal("migration:", err)
		}
	}
	const secret = "integration-test-secret"
	userService := users.NewService(users.NewPostgresRepository(pool), secret, time.Hour)
	chatService := chat.NewService(chat.NewPostgresRepository(pool))
	appConfig := config.Config{CORSOrigin: "https://localhost:3000"}
	hub := ws.NewHub()
	userHandler := users.NewHandler(userService, "", true, http.SameSiteLaxMode, time.Hour)
	userHandler.OnGuestDeleted = func(deleted users.DeletedGuest) {
		hub.Broadcast(ws.OutboundEvent{Type: ws.EventUserRemoved, UserID: deleted.UserID})
		hub.DisconnectUser(deleted.UserID)
	}
	googleHandler := users.NewGoogleHandler(userService, config.Config{JWTSecret: secret, FrontendURL: "https://frontend.test", CookieSecure: true, AccessTokenTTL: time.Hour})
	googleHandler.Provider = mockGoogleProvider{}
	router := httpapi.NewRouter(httpapi.Dependencies{Config: appConfig, AuthMiddleware: auth.Middleware(secret, userService.IsActive), Google: googleHandler, Users: userHandler, Chat: chat.NewHandler(chatService), WebSocket: ws.NewHandler(hub, ws.NewTicketStore(time.Minute), chatService, appConfig.CORSOrigin)})
	server := httptest.NewUnstartedServer(router)
	server.TLS = appConfig.TLSConfig()
	server.StartTLS()
	defer server.Close()
	client := func() *http.Client {
		c := server.Client()
		jar, _ := cookiejar.New(nil)
		return &http.Client{Transport: c.Transport, Jar: jar, Timeout: 10 * time.Second}
	}
	a, b, outsider := client(), client(), client()
	request := func(c *http.Client, method, path string, body any, want int, out any) {
		t.Helper()
		data, _ := json.Marshal(body)
		req, err := http.NewRequestWithContext(ctx, method, server.URL+path, bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		res, err := c.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != want {
			var result any
			_ = json.NewDecoder(res.Body).Decode(&result)
			t.Fatalf("%s %s: got %d want %d: %v", method, path, res.StatusCode, want, result)
		}
		if out != nil {
			if err := json.NewDecoder(res.Body).Decode(out); err != nil {
				t.Fatal(err)
			}
		}
	}
	request(client(), "GET", "/api/conversations", nil, 401, nil)
	var alice, bob, carol users.User
	request(a, "POST", "/api/auth/guest", map[string]string{"username": "Alice"}, 201, &alice)
	request(b, "POST", "/api/auth/guest", map[string]string{"username": "Bob"}, 201, &bob)
	request(outsider, "POST", "/api/auth/guest", map[string]string{"username": "Carol"}, 201, &carol)
	request(a, "GET", "/api/me", nil, 200, nil)
	request(client(), "POST", "/api/auth/guest", map[string]string{"username": "alice"}, 409, nil)
	var conversation, reverse chat.Conversation
	request(a, "POST", "/api/conversations", map[string]string{"peerId": bob.ID}, 201, &conversation)
	request(b, "POST", "/api/conversations", map[string]string{"peerId": alice.ID}, 201, &reverse)
	if conversation.ID != reverse.ID {
		t.Fatal("direct conversation duplicated")
	}
	request(a, "POST", "/api/conversations", map[string]string{"peerId": alice.ID}, 400, nil)
	request(a, "POST", "/api/conversations", map[string]string{"peerId": "invalid"}, 400, nil)
	request(a, "POST", "/api/conversations", map[string]string{"peerId": uuid.NewString()}, 404, nil)
	request(outsider, "GET", "/api/conversations/"+conversation.ID+"/messages", nil, 403, nil)
	connect := func(c *http.Client) *websocket.Conn {
		t.Helper()
		var ticket map[string]string
		request(c, "POST", "/api/ws/ticket", nil, 200, &ticket)
		conn, _, err := websocket.Dial(ctx, strings.Replace(server.URL, "https:", "wss:", 1)+"/ws?ticket="+ticket["ticket"], &websocket.DialOptions{HTTPClient: c, HTTPHeader: http.Header{"Origin": []string{appConfig.CORSOrigin}}})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.CloseNow() })
		return conn
	}
	aliceWS, bobWS, carolWS := connect(a), connect(b), connect(outsider)
	send := func(conn *websocket.Conn, event ws.InboundEvent) {
		t.Helper()
		if err := wsjson.Write(ctx, conn, event); err != nil {
			t.Fatal(err)
		}
	}
	receive := func(conn *websocket.Conn, kind ws.EventType) ws.OutboundEvent {
		t.Helper()
		readCtx, stop := context.WithTimeout(ctx, 15*time.Second)
		defer stop()
		for {
			var event ws.OutboundEvent
			if err := wsjson.Read(readCtx, conn, &event); err != nil {
				t.Fatal("websocket:", err)
			}
			if event.Type == kind {
				return event
			}
			if event.Type == ws.EventError && kind != ws.EventError {
				t.Fatal(event.Error)
			}
		}
	}
	messageID := uuid.NewString()
	input := ws.InboundEvent{Type: ws.EventMessageSend, ConversationID: conversation.ID, ClientMessageID: messageID, Content: "Hola, ¿cómo estás? 👋"}
	send(aliceWS, input)
	ack := receive(aliceWS, ws.EventMessageAck)
	// Bob receives a message even before joining/opening the conversation room.
	delivered := receive(bobWS, ws.EventMessageNew)
	if ack.Message == nil || delivered.Message == nil || ack.Message.ID != delivered.Message.ID || delivered.Message.Content != input.Content {
		t.Fatal("delivery mismatch")
	}
	send(aliceWS, input)
	retry := receive(aliceWS, ws.EventMessageAck)
	if retry.Message.ID != ack.Message.ID {
		t.Fatal("retry duplicated message")
	}
	var list []chat.Conversation
	request(b, "GET", "/api/conversations", nil, 200, &list)
	if len(list) != 1 || list[0].UnreadCount != 1 || list[0].Peer.ID != alice.ID {
		t.Fatalf("unexpected conversations: %+v", list)
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM messages").Scan(&count); err != nil || count != 1 {
		t.Fatal("duplicate persisted message", err, count)
	}
	send(carolWS, input)
	receive(carolWS, ws.EventError)
	send(bobWS, input)
	receive(bobWS, ws.EventError) // cannot claim someone else's clientMessageId
	if err := chatService.MarkRead(ctx, bob.ID, conversation.ID, ack.Message.ID); err != nil {
		t.Fatal("mark read:", err)
	}
	send(bobWS, ws.InboundEvent{Type: ws.EventMessageRead, ConversationID: conversation.ID, MessageID: ack.Message.ID})
	receive(aliceWS, ws.EventMessageRead)
	request(b, "GET", "/api/conversations", nil, 200, &list)
	if list[0].UnreadCount != 0 {
		t.Fatal("read count not cleared")
	}
	for i := 0; i < 3; i++ {
		input.ClientMessageID = uuid.NewString()
		input.Content = fmt.Sprintf("Mensaje %d", i)
		send(aliceWS, input)
		receive(aliceWS, ws.EventMessageAck)
	}
	var first, second chat.MessagePage
	request(b, "GET", "/api/conversations/"+conversation.ID+"/messages?limit=2", nil, 200, &first)
	request(b, "GET", "/api/conversations/"+conversation.ID+"/messages?limit=2&before="+first.NextCursor, nil, 200, &second)
	if len(first.Messages) != 2 || len(second.Messages) != 2 || first.NextCursor == "" || second.NextCursor != "" || second.Messages[0].ID != ack.Message.ID {
		t.Fatal("pagination mismatch")
	}
	request(b, "GET", "/api/conversations/"+conversation.ID+"/messages?before=garbage", nil, 400, nil)
	if err := chatService.MarkRead(ctx, bob.ID, conversation.ID, first.Messages[1].ID); err != nil {
		t.Fatal(err)
	}
	if err := chatService.MarkRead(ctx, bob.ID, conversation.ID, ack.Message.ID); err != nil {
		t.Fatal(err)
	}
	var lastRead string
	if err := pool.QueryRow(ctx, "SELECT last_read_message_id::text FROM conversation_members WHERE conversation_id=$1 AND user_id=$2", conversation.ID, bob.ID).Scan(&lastRead); err != nil || lastRead != first.Messages[1].ID {
		t.Fatal("read cursor moved backwards", err)
	}
	// Concurrent sends with one retry key persist once and increment unread once.
	var wg sync.WaitGroup
	failures := make(chan error, 8)
	concurrentID := uuid.NewString()
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := chatService.SendMessage(ctx, alice.ID, conversation.ID, concurrentID, "Concurrente")
			failures <- err
		}()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	request(b, "GET", "/api/conversations", nil, 200, &list)
	if list[0].UnreadCount != 1 {
		t.Fatal("concurrent retry changed unread count", list[0].UnreadCount)
	}

	// A Google account survives logout and is recovered using its immutable subject.
	googleClient := client()
	googleClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	googleSignIn := func(c *http.Client) users.User {
		t.Helper()
		res, err := c.Get(server.URL + "/api/auth/google")
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != 302 {
			t.Fatal("Google start failed", res.StatusCode)
		}
		location, err := url.Parse(res.Header.Get("Location"))
		if err != nil {
			t.Fatal(err)
		}
		res, err = c.Get(server.URL + "/api/auth/google/callback?code=valid&state=" + url.QueryEscape(location.Query().Get("state")))
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != 303 || res.Header.Get("Location") != "https://frontend.test/chat" {
			t.Fatal("Google callback failed", res.StatusCode, res.Header.Get("Location"))
		}
		persistent := false
		for _, cookie := range res.Cookies() {
			if cookie.Name == auth.CookieName {
				persistent = cookie.MaxAge > 0 && cookie.HttpOnly && cookie.Secure
			}
		}
		if !persistent {
			t.Fatal("Google cookie must persist securely")
		}
		var user users.User
		request(c, "GET", "/api/me", nil, 200, &user)
		return user
	}
	googleUser := googleSignIn(googleClient)
	if googleUser.IsGuest || googleUser.AuthProvider != "google" || googleUser.ID == alice.ID {
		t.Fatal("Google identity was not isolated from a matching guest nickname")
	}
	var mixed chat.Conversation
	request(googleClient, "POST", "/api/conversations", map[string]string{"peerId": carol.ID}, 201, &mixed)
	permanent, err := chatService.SendMessage(ctx, googleUser.ID, mixed.ID, uuid.NewString(), "Mensaje de Google que se conserva")
	if err != nil {
		t.Fatal(err)
	}
	temporary, err := chatService.SendMessage(ctx, carol.ID, mixed.ID, uuid.NewString(), "Mensaje de invitado que se elimina")
	if err != nil {
		t.Fatal(err)
	}
	if err = chatService.MarkRead(ctx, googleUser.ID, mixed.ID, temporary.ID); err != nil {
		t.Fatal(err)
	}
	request(outsider, "POST", "/api/auth/logout", nil, 204, nil)
	request(outsider, "GET", "/api/me", nil, 401, nil)
	var remaining chat.MessagePage
	request(googleClient, "GET", "/api/conversations/"+mixed.ID+"/messages", nil, 200, &remaining)
	if len(remaining.Messages) != 1 || remaining.Messages[0].ID != permanent.ID {
		t.Fatal("guest deletion removed another participant's messages")
	}
	request(googleClient, "GET", "/api/conversations", nil, 200, &list)
	if len(list) != 1 || list[0].Peer.ID != "" || list[0].UnreadCount != 0 {
		t.Fatal("closed guest chat did not retain the correct history", list)
	}
	if _, err = chatService.SendMessage(ctx, googleUser.ID, mixed.ID, uuid.NewString(), "No enviar a un invitado eliminado"); !errors.Is(err, chat.ErrClosed) {
		t.Fatal("closed chat still accepts messages", err)
	}
	request(googleClient, "POST", "/api/auth/logout", nil, 204, nil)
	recovered := googleSignIn(googleClient)
	if recovered.ID != googleUser.ID {
		t.Fatal("Google sign-in created another account")
	}
	request(googleClient, "GET", "/api/conversations/"+mixed.ID+"/messages", nil, 200, &remaining)
	if len(remaining.Messages) != 1 {
		t.Fatal("Google history lost after signing in again")
	}
	// Nicknames can be reused, but the previous guest's token never gains access.
	var newCarol users.User
	freshGuest := client()
	request(freshGuest, "POST", "/api/auth/guest", map[string]string{"username": "Carol"}, 201, &newCarol)
	if newCarol.ID == carol.ID {
		t.Fatal("deleted guest identity was recovered")
	}
	if active, err := userService.IsActive(ctx, carol.ID); err != nil || active {
		t.Fatal("deleted guest token remains active", err)
	}
	var directory []users.User
	request(freshGuest, "GET", "/api/users", nil, 200, &directory)
	foundGoogle := false
	for _, user := range directory {
		if user.Email != "" {
			t.Fatal("directory exposes email")
		}
		if user.ID == googleUser.ID {
			foundGoogle = user.AuthProvider == "google"
		}
	}
	if !foundGoogle {
		t.Fatal("guest cannot distinguish Google users")
	}
	if _, err = users.NewPostgresRepository(pool).DeleteGuest(ctx, googleUser.ID, false); !errors.Is(err, users.ErrNotFound) {
		t.Fatal("guest cleanup accepted a permanent account", err)
	}
	// Expiration removes abandoned guests even if no explicit logout was received.
	if _, err = pool.Exec(ctx, "UPDATE users SET guest_expires_at=now()-interval '1 minute' WHERE id=$1", newCarol.ID); err != nil {
		t.Fatal(err)
	}
	if err = userService.ExpireGuests(ctx, userHandler.OnGuestDeleted); err != nil {
		t.Fatal(err)
	}
	if active, err := userService.IsActive(ctx, newCarol.ID); err != nil || active {
		t.Fatal("expired guest still active", err)
	}
	t.Log("HTTPS/WSS, persistence, Google recovery, badges, guest deletion and session expiration verified")
}

type mockGoogleProvider struct{}

func (mockGoogleProvider) AuthorizationURL(state, nonce, verifier string) string {
	return "https://google.test/authorize?state=" + url.QueryEscape(state)
}
func (mockGoogleProvider) Exchange(_ context.Context, code, nonce, verifier string) (users.GoogleIdentity, error) {
	if code != "valid" || nonce == "" || verifier == "" {
		return users.GoogleIdentity{}, errors.New("invalid mock authorization")
	}
	return users.GoogleIdentity{Subject: "google-integration-subject", Email: "google-user@example.test", Name: "Alice"}, nil
}
