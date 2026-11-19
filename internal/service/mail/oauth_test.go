package mail

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"crm.local/internal/model"
	"crm.local/internal/permissions"
	"crm.local/internal/repository"
	"crm.local/internal/security"
)

// mockProviderServer emulates the provider's token + userinfo endpoints.
func mockProviderServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", 400)
			return
		}
		if r.PostForm.Get("grant_type") == "authorization_code" && r.PostForm.Get("code") != "good-code" {
			w.WriteHeader(400)
			fmt.Fprint(w, `{"error":"invalid_grant","error_description":"bad code"}`)
			return
		}
		if r.PostForm.Get("grant_type") == "refresh_token" && r.PostForm.Get("refresh_token") != "rt-1" {
			w.WriteHeader(400)
			fmt.Fprint(w, `{"error":"invalid_grant"}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"at-1","refresh_token":"rt-1","token_type":"Bearer","expires_in":3600,"scope":"mail"}`)
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer at-1" {
			w.WriteHeader(401)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"email":"oauth-user@example.com"}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func newOAuthTestService(t *testing.T, db *sql.DB, base string) *Service {
	t.Helper()
	repo := repository.NewMailRepository(db)
	allow := false // tests use mock hosts; skip SSRF host blocking
	return New(repo, nil, nil, Config{
		TrackingBaseURL:   base,
		EncryptionKey:     testKey,
		PollInterval:      time.Minute,
		FetchLimit:        10,
		BlockPrivateHosts: &allow,
	}, slog.Default())
}

func TestOAuthConnectFlow(t *testing.T) {
	db := openTestDB(t)
	mock := mockProviderServer(t)
	restore := SetProvidersForTest([]Provider{{
		ID:            "google",
		Name:          "Google (Gmail)",
		AuthEndpoint:  mock.URL + "/auth",
		TokenEndpoint: mock.URL + "/token",
		UserInfoEndpoint: mock.URL + "/userinfo",
		IMAPHost:      "imap.gmail.com",
		IMAPPort:      993,
		Scopes:        []string{"https://mail.google.com/"},
		ClientID:      "cid",
		ClientSecret:  "csecret",
	}})
	defer restore()

	svc := newOAuthTestService(t, db, "http://crm.test")
	ctx := context.Background()

	// 1) begin connect -> authorization URL contains state + client_id + redirect_uri
	authURL, err := svc.BeginConnect(ctx, "ws1", "u1", "google")
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	state := q.Get("state")
	if state == "" || q.Get("client_id") != "cid" || q.Get("redirect_uri") != "http://crm.test/api/v1/mail/oauth/callback" {
		t.Fatalf("bad auth url: %s", authURL)
	}
	if q.Get("scope") != "https://mail.google.com/" || q.Get("access_type") != "offline" {
		t.Fatalf("bad scope/access_type: %s", authURL)
	}

	// 2) unconfigured provider must error
	if _, err := svc.BeginConnect(ctx, "ws1", "u1", "microsoft"); err == nil {
		t.Fatal("expected error for unconfigured provider")
	}

	// 3) complete with bad code -> provider error surfaces
	if _, err := svc.CompleteConnect(ctx, "ws1", "u1", "google", state, "bad-code"); err == nil {
		t.Fatal("expected invalid_grant error")
	}
	// note: the failed attempt consumed the state, so re-begin
	authURL, err = svc.BeginConnect(ctx, "ws1", "u1", "google")
	if err != nil {
		t.Fatal(err)
	}
	state = mustState(t, authURL)

	// 4) complete with good code -> connection created with sealed tokens
	conn, err := svc.CompleteConnect(ctx, "ws1", "u1", "google", state, "good-code")
	if err != nil {
		t.Fatal(err)
	}
	if conn.AuthMode != "oauth" || conn.OAuthProvider != "google" || conn.Provider != "gmail" {
		t.Fatalf("bad connection: %+v", conn)
	}
	if conn.EmailAddress != "oauth-user@example.com" {
		t.Fatalf("email not resolved: %q", conn.EmailAddress)
	}
	if conn.IMAPHost != "imap.gmail.com" || !conn.UseTLS || !conn.Enabled {
		t.Fatalf("imap defaults wrong: %+v", conn)
	}
	if conn.OAuthSealed != "" || conn.PasswordSealed != "" {
		t.Fatal("secrets leaked in response")
	}

	// 5) state is single-use: replay must fail
	if _, err := svc.CompleteConnect(ctx, "ws1", "u1", "google", state, "good-code"); err == nil {
		t.Fatal("state replay succeeded")
	}

	// 6) tokens are sealed at rest and unseal to the right values
	stored, err := svc.repo.GetConnection(ctx, conn.ID)
	if err != nil || stored == nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stored.OAuthSealed, "enc:v1:") {
		t.Fatalf("oauth tokens not sealed: %q", stored.OAuthSealed)
	}
	toks, err := svc.unsealOAuth(stored)
	if err != nil || toks.AccessToken != "at-1" || toks.RefreshToken != "rt-1" {
		t.Fatalf("unseal failed: %v %+v", err, toks)
	}

	// 7) accessToken returns the token; refresh rotates it
	tok, err := svc.accessToken(ctx, stored)
	if err != nil || tok != "at-1" {
		t.Fatalf("accessToken: %v %q", err, tok)
	}
	if err := svc.RefreshAccessToken(ctx, stored); err != nil {
		t.Fatal(err)
	}
	stored2, _ := svc.repo.GetConnection(ctx, conn.ID)
	toks2, err := svc.unsealOAuth(stored2)
	if err != nil || toks2.AccessToken != "at-1" || toks2.RefreshToken != "rt-1" {
		t.Fatalf("refresh did not persist: %v %+v", err, toks2)
	}
}

func mustState(t *testing.T, authURL string) string {
	t.Helper()
	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatal(err)
	}
	s := u.Query().Get("state")
	if s == "" {
		t.Fatal("no state in url")
	}
	return s
}

func TestOAuthStateWorkspaceIsolation(t *testing.T) {
	db := openTestDB(t)
	svc := newOAuthTestService(t, db, "http://crm.test")
	ctx := context.Background()
	// seed a state directly for ws1
	st := mustCreateState(t, svc, "ws1", "u1", "google")
	// ws2 must not resolve it
	if _, _, err := svc.ResolveState(ctx, st); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.repo.ConsumeOAuthState(ctx, st, "ws2"); err == nil {
		t.Fatal("cross-workspace state consumption succeeded")
	}
}

func mustCreateState(t *testing.T, svc *Service, ws, user, provider string) string {
	t.Helper()
	state := "test-state-0123456789"
	err := svc.repo.CreateOAuthState(context.Background(), &model.MailOAuthState{
		State: state, WorkspaceID: ws, Provider: provider, CreatedBy: user,
		ExpiresAt: time.Now().UTC().Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	return state
}

var _ = permissions.AuthContext{}
var _ = security.SealCredentials
