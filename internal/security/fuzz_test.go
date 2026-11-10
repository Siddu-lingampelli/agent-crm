// internal/security/fuzz_test.go
package security

import (
	"testing"
)

func FuzzValidateSSRFURL(f *testing.F) {
	// Seed corpus
	testCases := []string{
		"http://example.com",
		"https://example.com/path",
		"http://localhost:8080",
		"http://127.0.0.1",
		"http://10.0.0.1",
		"http://192.168.1.1",
		"http://172.16.0.1",
		"http://[::1]",
		"file:///etc/passwd",
		"ftp://example.com",
		"http://example.com@evil.com",
		"http://example.com:80@evil.com",
		"http://169.254.169.254/latest/meta-data/",
		"http://metadata.google.internal",
	}

	for _, tc := range testCases {
		f.Add(tc)
	}

	f.Fuzz(func(t *testing.T, url string) {
		config := SecurityConfig{
			SSRFBlockPrivateIPs: true,
			SSRFAllowedCIDRs:    []string{},
		}
		// Should not panic
		_ = ValidateSSRFURL(url, config)
	})
}

func FuzzRateLimitKey(f *testing.F) {
	f.Add("192.168.1.1")
	f.Add("10.0.0.1")
	f.Add("2001:db8::1")

	f.Fuzz(func(t *testing.T, ip string) {
		// Test rate limiter key extraction doesn't panic
		rl := newRateLimiter(DefaultSecurityConfig())
		_ = rl.getLimiter(ip)
	})
}
