package mail

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"crm.local/internal/model"
	"crm.local/internal/permissions"
	"crm.local/internal/repository"
	"crm.local/internal/security"
)

// openTestDB creates a temp SQLite DB with the mail tables from migration 00012
// plus the minimal workspaces table needed for FK references.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS workspaces (id TEXT PRIMARY KEY); INSERT INTO workspaces (id) VALUES ('ws1');`); err != nil {
		t.Fatal(err)
	}
	// emails table needed for email_tracking FK
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS emails (id TEXT PRIMARY KEY);`); err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../../../db/migrations/00012_mail_connections_inbox_tracking.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range splitSQL(string(migration)) {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("exec migration stmt: %v\n%s", err, s)
		}
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func splitSQL(mig string) []string {
	up := mig
	if i := strings.Index(up, "-- +goose Down"); i >= 0 {
		up = up[:i]
	}
	parts := []string{}
	cur := ""
	for _, line := range strings.Split(up, "\n") {
		if line == "-- +goose StatementBegin" || line == "-- +goose StatementEnd" {
			continue
		}
		cur += line + "\n"
		if line != "" && line[len(line)-1] == ';' {
			parts = append(parts, cur)
			cur = ""
		}
	}
	if cur != "" {
		parts = append(parts, cur)
	}
	return parts
}

const testKey = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"

func testActor() permissions.AuthContext {
	return permissions.AuthContext{
		ActorType: "user", ActorID: "u1", WorkspaceID: "ws1",
		Scopes: []string{"integrations:read", "integrations:write", "emails:read", "emails:write"},
	}
}

func newTestService(t *testing.T, db *sql.DB) *Service {
	t.Helper()
	repo := repository.NewMailRepository(db)
	allow := false // tests use fake hosts; skip SSRF host blocking
	return New(repo, nil, nil, Config{
		TrackingBaseURL:   "http://crm.test",
		EncryptionKey:     testKey,
		PollInterval:      0,
		FetchLimit:        10,
		BlockPrivateHosts: &allow,
	}, slog.Default())
}

func TestCreateConnectionSealsPassword(t *testing.T) {
	db := openTestDB(t)
	svc := newTestService(t, db)
	c, err := svc.CreateConnection(context.Background(), testActor(), CreateConnectionInput{
		Name: "Work inbox", EmailAddress: "me@example.com", IMAPHost: "imap.example.com",
		Username: "me@example.com", Password: "s3cret", PollIntervalSecs: 30,
	})
	if err != nil {
		t.Fatal(err)
	}
	if c.PasswordSealed != "" {
		t.Fatal("sealed password leaked in response")
	}
	if c.IMAPPort != 993 || !c.UseTLS || c.Mailbox != "INBOX" {
		t.Fatalf("defaults not applied: %+v", c)
	}
	// stored row must be sealed and unsealable
	stored, err := svc.repo.GetConnection(context.Background(), c.ID)
	if err != nil || stored == nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stored.PasswordSealed, "enc:v1:") {
		t.Fatalf("password not sealed: %q", stored.PasswordSealed)
	}
	pw, err := security.UnsealCredentials(stored.PasswordSealed, testKey)
	if err != nil || pw != "s3cret" {
		t.Fatalf("unseal failed: %v %q", err, pw)
	}
	// wrong key must fail
	if _, err := security.UnsealCredentials(stored.PasswordSealed, strings.Repeat("ab", 32)); err == nil {
		t.Fatal("unseal with wrong key succeeded")
	}
}

func TestCreateConnectionValidation(t *testing.T) {
	db := openTestDB(t)
	svc := newTestService(t, db)
	cases := []CreateConnectionInput{
		{EmailAddress: "me@example.com", IMAPHost: "h", Password: "p"},                                        // missing name
		{Name: "n", EmailAddress: "not-an-email", IMAPHost: "h", Password: "p"},                              // bad email
		{Name: "n", EmailAddress: "me@example.com", Password: "p"},                                            // missing host
		{Name: "n", EmailAddress: "me@example.com", IMAPHost: "h"},                                            // missing password
	}
	for i, in := range cases {
		if _, err := svc.CreateConnection(context.Background(), testActor(), in); err == nil {
			t.Fatalf("case %d: expected error", i)
		}
	}
}

func TestUpdateConnectionKeepsPasswordWhenEmpty(t *testing.T) {
	db := openTestDB(t)
	svc := newTestService(t, db)
	c, err := svc.CreateConnection(context.Background(), testActor(), CreateConnectionInput{
		Name: "n", EmailAddress: "me@example.com", IMAPHost: "h", Password: "pw1",
	})
	if err != nil {
		t.Fatal(err)
	}
	disabled := false
	upd, err := svc.UpdateConnection(context.Background(), testActor(), c.ID, UpdateConnectionInput{
		Name: "renamed", Enabled: &disabled,
	})
	if err != nil {
		t.Fatal(err)
	}
	if upd.Name != "renamed" || upd.Enabled {
		t.Fatalf("update not applied: %+v", upd)
	}
	stored, _ := svc.repo.GetConnection(context.Background(), c.ID)
	pw, err := security.UnsealCredentials(stored.PasswordSealed, testKey)
	if err != nil || pw != "pw1" {
		t.Fatal("password changed when not supplied")
	}
	// now rotate password
	if _, err := svc.UpdateConnection(context.Background(), testActor(), c.ID, UpdateConnectionInput{Password: "pw2"}); err != nil {
		t.Fatal(err)
	}
	stored, _ = svc.repo.GetConnection(context.Background(), c.ID)
	pw, _ = security.UnsealCredentials(stored.PasswordSealed, testKey)
	if pw != "pw2" {
		t.Fatalf("password rotation failed: %q", pw)
	}
}

func TestTrackingLifecycle(t *testing.T) {
	db := openTestDB(t)
	svc := newTestService(t, db)
	ctx := context.Background()
	if _, err := db.Exec(`INSERT INTO emails (id) VALUES ('em1')`); err != nil {
		t.Fatal(err)
	}
	tok, err := svc.EnsureTracking(ctx, "ws1", "em1")
	if err != nil {
		t.Fatal(err)
	}
	if tok == "" {
		t.Fatal("empty token")
	}
	// idempotent
	tok2, _ := svc.EnsureTracking(ctx, "ws1", "em1")
	if tok2 != tok {
		t.Fatal("EnsureTracking not idempotent")
	}
	if u := svc.TrackingPixelURL(tok); u != "http://crm.test/t/o/"+tok+".gif" {
		t.Fatalf("bad pixel url: %s", u)
	}
	click := svc.TrackingClickURL(tok, "https://example.com/pricing")
	if !strings.HasPrefix(click, "http://crm.test/t/c/"+tok+"?u=") {
		t.Fatalf("bad click url: %s", click)
	}
	// record events
	if err := svc.RecordEvent(ctx, tok, "open", "", "1.2.3.4", "UA"); err != nil {
		t.Fatal(err)
	}
	if err := svc.RecordEvent(ctx, tok, "open", "", "1.2.3.4", "UA"); err != nil {
		t.Fatal(err)
	}
	if err := svc.RecordEvent(ctx, tok, "click", "https://example.com/pricing", "1.2.3.4", "UA"); err != nil {
		t.Fatal(err)
	}
	if err := svc.RecordEvent(ctx, tok, "delivered", "", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := svc.RecordEvent(ctx, tok, "bounced", "550 no such user", "", ""); err != nil {
		t.Fatal(err)
	}
	tr, events, err := svc.GetTracking(ctx, testActor(), "em1")
	if err != nil {
		t.Fatal(err)
	}
	if tr.OpenCount != 2 || tr.ClickCount != 1 || !tr.Delivered || !tr.Bounced {
		t.Fatalf("bad aggregate: %+v", tr)
	}
	if tr.BounceReason != "550 no such user" {
		t.Fatalf("bad bounce reason: %q", tr.BounceReason)
	}
	if len(events) != 5 {
		t.Fatalf("expected 5 events, got %d", len(events))
	}
	// unknown token must error
	if err := svc.RecordEvent(ctx, "nope", "open", "", "", ""); err == nil {
		t.Fatal("unknown token accepted")
	}
}

func TestRewriteLinksAndPixel(t *testing.T) {
	db := openTestDB(t)
	svc := newTestService(t, db)
	html := `<p>Hi</p><a href="https://example.com/a">A</a><a href='http://example.com/b'>B</a>`
	out := svc.RewriteLinks("tok123", html)
	if !strings.Contains(out, "http://crm.test/t/c/tok123?u=") {
		t.Fatalf("links not rewritten: %s", out)
	}
	if strings.Contains(out, `href="https://example.com/a"`) {
		t.Fatal("original href survived")
	}
	withPixel := svc.InjectPixel("<html><body>x</body></html>", "tok123")
	if !strings.Contains(withPixel, "/t/o/tok123.gif") || !strings.Contains(withPixel, "</body>") {
		t.Fatalf("pixel not injected before </body>: %s", withPixel)
	}
	noBody := svc.InjectPixel("<p>x</p>", "tok123")
	if !strings.HasSuffix(noBody, `"/>`) {
		t.Fatalf("pixel not appended: %s", noBody)
	}
}

func TestInboundListAndFlags(t *testing.T) {
	db := openTestDB(t)
	svc := newTestService(t, db)
	ctx := context.Background()
	repo := svc.repo
	conn := insertConn(t, repo)
	m := insertInbound(t, repo, conn.ID, 42, "alice@example.com")
	page, err := svc.ListInbound(ctx, testActor(), repository.InboundListOptions{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || page.Items[0].ID != m.ID {
		t.Fatalf("list failed: %+v", page)
	}
	read := true
	upd, err := svc.UpdateInboundFlags(ctx, testActor(), m.ID, UpdateInboundFlagsInput{IsRead: &read, RelatedType: "contact", RelatedID: "c9"})
	if err != nil {
		t.Fatal(err)
	}
	if !upd.IsRead || upd.RelatedType != "contact" || upd.RelatedID != "c9" {
		t.Fatalf("flags not updated: %+v", upd)
	}
	page, err = svc.ListInbound(ctx, testActor(), repository.InboundListOptions{IsRead: &read, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 {
		t.Fatalf("is_read=true filter failed: %+v", page)
	}
	unread := false
	page, err = svc.ListInbound(ctx, testActor(), repository.InboundListOptions{IsRead: &unread, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 0 {
		t.Fatalf("is_read=false filter should be empty: %+v", page)
	}
}

func insertConn(t *testing.T, repo *repository.MailRepository) *model.MailConnection {
	t.Helper()
	sealed, err := security.SealCredentials("pw", testKey)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	c := &model.MailConnection{
		ID: "conn1", WorkspaceID: "ws1", Name: "n", Provider: "imap", EmailAddress: "me@example.com",
		IMAPHost: "imap.example.com", IMAPPort: 993, UseTLS: true, Username: "me@example.com",
		PasswordSealed: sealed, Mailbox: "INBOX", Enabled: true, PollIntervalSecs: 60,
		CreatedBy: "u1", CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateConnection(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	return c
}

func insertInbound(t *testing.T, repo *repository.MailRepository, connID string, uid int, from string) *model.InboundMessage {
	t.Helper()
	now := time.Now().UTC()
	m := &model.InboundMessage{
		ID: "im1", WorkspaceID: "ws1", ConnectionID: connID, MessageID: "<m1@example.com>",
		UID: uid, ThreadKey: "<m1@example.com>", FromAddr: from, FromName: "Alice",
		ToAddr: "me@example.com", Subject: "Hello", BodyText: "hi there", Snippet: "hi there",
		ReceivedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	inserted, err := repo.InsertInbound(context.Background(), m)
	if err != nil || !inserted {
		t.Fatalf("insert inbound: %v %v", inserted, err)
	}
	// duplicate (same conn+uid) must be ignored
	dup, err := repo.InsertInbound(context.Background(), m)
	if err != nil || dup {
		t.Fatalf("duplicate insert should be ignored: %v %v", dup, err)
	}
	return m
}
