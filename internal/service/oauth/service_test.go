package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"testing"
	"time"

	"crm.local/internal/permissions"
	"crm.local/internal/repository"
)

// newTestService builds a service backed by an in-memory SQLite DB with the
// oauth tables created inline.
func newTestService(t *testing.T) *Service {
	t.Helper()
	db, err := openTestDB(t)
	if err != nil {
		t.Fatal(err)
	}
	repo := repository.NewOAuthRepository(db)
	return New(repo, "http://test.local", []byte("test-secret"))
}

func TestAuthorizeRequiresPKCE(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	reg, err := svc.RegisterClient(ctx, RegisterClientInput{
		ClientName:   "test",
		RedirectURIs: []string{"http://client.local/cb"},
		Scopes:       []string{"leads:read", "leads:write"},
	}, "ws1", false)
	if err != nil {
		t.Fatal(err)
	}
	// missing PKCE -> error
	_, err = svc.Authorize(ctx, AuthorizeInput{
		ClientID: reg.ClientID, RedirectURI: "http://client.local/cb",
		ResponseType: "code", WorkspaceID: "ws1", UserID: "u1",
	})
	if err == nil {
		t.Fatal("expected PKCE error")
	}
	// with PKCE -> redirect with code
	verifier := "test-verifier-0123456789012345678901234567890123"
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	redirect, err := svc.Authorize(ctx, AuthorizeInput{
		ClientID: reg.ClientID, RedirectURI: "http://client.local/cb",
		ResponseType: "code", CodeChallenge: challenge, CodeChallengeMethod: "S256",
		WorkspaceID: "ws1", UserID: "u1", State: "xyz",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(redirect) < 10 || redirect[:4] != "http" {
		t.Fatalf("bad redirect: %q", redirect)
	}
}

func TestTokenExchangeAndValidation(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	reg, err := svc.RegisterClient(ctx, RegisterClientInput{
		ClientName:   "test",
		RedirectURIs: []string{"http://client.local/cb"},
		Scopes:       []string{"leads:read", "leads:write"},
	}, "ws1", false)
	if err != nil {
		t.Fatal(err)
	}
	verifier := "v123456789012345678901234567890123456789012345678901234"
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	redirect, err := svc.Authorize(ctx, AuthorizeInput{
		ClientID: reg.ClientID, RedirectURI: "http://client.local/cb",
		ResponseType: "code", CodeChallenge: challenge, CodeChallengeMethod: "S256",
		Scope: "leads:read", WorkspaceID: "ws1", UserID: "u1",
	})
	if err != nil {
		t.Fatal(err)
	}
	// extract code
	code := ""
	for _, part := range splitQuery(redirect) {
		if len(part) > 5 && part[:5] == "code=" {
			code = part[5:]
		}
	}
	if code == "" {
		t.Fatalf("no code in %q", redirect)
	}
	// codes are URL-encoded in the redirect; decode percent-encoding
	if decoded, err := url.QueryUnescape(code); err == nil {
		code = decoded
	}
	// first exchange with the RIGHT verifier must succeed
	tok, err := svc.Token(ctx, TokenInput{
		GrantType: "authorization_code", Code: code, RedirectURI: "http://client.local/cb",
		ClientID: reg.ClientID, CodeVerifier: verifier,
	})
	if err != nil {
		t.Fatal(err)
	}
	// wrong verifier against a fresh code must fail PKCE
	verifier2 := "w123456789012345678901234567890123456789012345678901234"
	sum2 := sha256.Sum256([]byte(verifier2))
	challenge2 := base64.RawURLEncoding.EncodeToString(sum2[:])
	redirect2, err := svc.Authorize(ctx, AuthorizeInput{
		ClientID: reg.ClientID, RedirectURI: "http://client.local/cb",
		ResponseType: "code", CodeChallenge: challenge2, CodeChallengeMethod: "S256",
		Scope: "leads:read", WorkspaceID: "ws1", UserID: "u1",
	})
	if err != nil {
		t.Fatal(err)
	}
	code2 := ""
	for _, part := range splitQuery(redirect2) {
		if len(part) > 5 && part[:5] == "code=" {
			code2 = part[5:]
		}
	}
	if decoded, derr := url.QueryUnescape(code2); derr == nil {
		code2 = decoded
	}
	_, err = svc.Token(ctx, TokenInput{
		GrantType: "authorization_code", Code: code2, RedirectURI: "http://client.local/cb",
		ClientID: reg.ClientID, CodeVerifier: "wrong-verifier",
	})
	if err == nil {
		t.Fatal("expected PKCE failure")
	}
	if tok.AccessToken == "" || tok.TokenType != "Bearer" {
		t.Fatalf("bad token response: %+v", tok)
	}
	// validate the JWT
	actx, err := svc.ValidateAccessToken(ctx, tok.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	if actx.ActorID != "u1" || actx.WorkspaceID != "ws1" {
		t.Fatalf("bad actor: %+v", actx)
	}
	if !permissions.HasScope(actx.Scopes, "leads:read") {
		t.Fatalf("scope missing: %v", actx.Scopes)
	}
	// tampered token -> error
	bad := tok.AccessToken[:len(tok.AccessToken)-2] + "xx"
	if _, err := svc.ValidateAccessToken(ctx, bad); err == nil {
		t.Fatal("expected signature failure")
	}
	// code is single-use
	_, err = svc.Token(ctx, TokenInput{
		GrantType: "authorization_code", Code: code, RedirectURI: "http://client.local/cb",
		ClientID: reg.ClientID, CodeVerifier: verifier,
	})
	if err == nil {
		t.Fatal("code should be single-use")
	}
}

func TestMetadata(t *testing.T) {
	svc := newTestService(t)
	prm := svc.ProtectedResourceMetadata()
	if prm["resource"] != "http://test.local/mcp" {
		t.Fatalf("bad resource: %v", prm["resource"])
	}
	asm := svc.AuthorizationServerMetadata()
	if asm["issuer"] != "http://test.local" {
		t.Fatalf("bad issuer: %v", asm["issuer"])
	}
	methods, ok := asm["code_challenge_methods_supported"].([]string)
	if !ok || len(methods) != 1 || methods[0] != "S256" {
		t.Fatalf("PKCE S256 must be the only method: %v", asm["code_challenge_methods_supported"])
	}
}

func splitQuery(u string) []string {
	out := []string{}
	cur := ""
	for _, c := range u {
		if c == '?' || c == '&' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
			continue
		}
		cur += string(c)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

var _ = time.Now
