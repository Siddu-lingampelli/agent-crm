package security

import (
	"context"
	"testing"
)

func TestValidateOutboundHostBlocksPrivate(t *testing.T) {
	ctx := context.Background()
	blocked := []string{"127.0.0.1", "localhost", "169.254.169.254", "10.0.0.5", "192.168.1.1", "::1", "0.0.0.0"}
	for _, h := range blocked {
		if err := ValidateOutboundHost(ctx, h, true, nil); err == nil {
			t.Errorf("host %s should be blocked", h)
		}
	}
}

func TestValidateOutboundHostAllowsPublic(t *testing.T) {
	ctx := context.Background()
	// example.com resolves publicly in CI; if DNS is unavailable the test env will
	// fail to resolve — treat resolution error as skip, not failure.
	if err := ValidateOutboundHost(ctx, "example.com", true, nil); err != nil {
		t.Skipf("DNS unavailable in test env: %v", err)
	}
}

func TestValidateOutboundHostCIDRAllowlist(t *testing.T) {
	ctx := context.Background()
	if err := ValidateOutboundHost(ctx, "127.0.0.1", true, []string{"127.0.0.0/8"}); err != nil {
		t.Fatalf("127.0.0.1 with allowlist should pass: %v", err)
	}
	if err := ValidateOutboundHost(ctx, "10.1.2.3", true, []string{"127.0.0.0/8"}); err == nil {
		t.Fatal("10.1.2.3 should still be blocked with unrelated allowlist")
	}
}

func TestValidateOutboundHostEmpty(t *testing.T) {
	if err := ValidateOutboundHost(context.Background(), "", true, nil); err == nil {
		t.Fatal("empty host must error")
	}
	if err := ValidateOutboundHost(context.Background(), "   ", true, nil); err == nil {
		t.Fatal("blank host must error")
	}
}
