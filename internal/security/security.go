// Package security implements security middleware for the HTTP server.
package security

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// SecurityConfig holds security-related configuration.
type SecurityConfig struct {
	// Rate limiting
	RateLimitEnabled bool
	RateLimitRPS     float64 // Requests per second
	RateLimitBurst   int     // Burst allowance

	// Request size limits
	MaxRequestBodySize int64 // bytes
	MaxUploadSize      int64 // bytes

	// CORS
	CORSAllowedOrigins   []string
	CORSAllowedMethods   []string
	CORSAllowedHeaders   []string
	CORSAllowCredentials bool

	// SSRF protection
	SSRFBlockPrivateIPs bool
	SSRFAllowedCIDRs    []string

	// Timeouts
	RequestTimeout time.Duration
}

// DefaultSecurityConfig returns a production-ready security config.
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		RateLimitEnabled:     true,
		RateLimitRPS:         100,              // 100 requests per second
		RateLimitBurst:       200,              // burst of 200
		MaxRequestBodySize:   10 * 1024 * 1024, // 10 MB
		MaxUploadSize:        50 * 1024 * 1024, // 50 MB
		CORSAllowedOrigins:   []string{"http://localhost:5173", "http://localhost:5174", "http://localhost:5175"},
		CORSAllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		CORSAllowedHeaders:   []string{"Content-Type", "Authorization", "X-Requested-With", "X-CSRF-Token", "Idempotency-Key"},
		CORSAllowCredentials: true,
		SSRFBlockPrivateIPs:  true,
		RequestTimeout:       30 * time.Second,
	}
}

// rateLimiter holds rate limiters per IP/client
type rateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	config   SecurityConfig
	lastSeen map[string]time.Time
}

func newRateLimiter(config SecurityConfig) *rateLimiter {
	return &rateLimiter{
		limiters: make(map[string]*rate.Limiter),
		config:   config,
		lastSeen: make(map[string]time.Time),
	}
}

func (rl *rateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	if len(rl.limiters) >= 10000 {
		for key, seen := range rl.lastSeen {
			if now.Sub(seen) > 5*time.Minute {
				delete(rl.limiters, key)
				delete(rl.lastSeen, key)
			}
		}
		// Use one shared overflow bucket instead of allocating without a bound.
		if _, exists := rl.limiters[key]; !exists {
			key = "overflow"
		}
	}
	rl.lastSeen[key] = now

	limiter, exists := rl.limiters[key]
	if !exists {
		limiter = rate.NewLimiter(rate.Limit(rl.config.RateLimitRPS), rl.config.RateLimitBurst)
		rl.limiters[key] = limiter
	}
	return limiter
}

// RateLimitMiddleware returns a middleware that enforces rate limiting per IP.
func RateLimitMiddleware(config SecurityConfig, log *slog.Logger) func(http.Handler) http.Handler {
	if !config.RateLimitEnabled {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	limiter := newRateLimiter(config)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract client IP
			clientIP := getClientIP(r)

			// Get rate limiter for this IP
			rl := limiter.getLimiter(clientIP)

			// Check if request is allowed
			if !rl.Allow() {
				log.Warn("rate limit exceeded", "ip", clientIP, "path", r.URL.Path)
				w.Header().Set("Retry-After", "1")
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequestSizeMiddleware limits the request body size.
func RequestSizeMiddleware(config SecurityConfig, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check Content-Length header
			if r.ContentLength > config.MaxRequestBodySize {
				log.Warn("request body too large", "content_length", r.ContentLength, "max", config.MaxRequestBodySize, "ip", getClientIP(r))
				http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
				return
			}

			// Wrap body reader with a limit reader
			r.Body = http.MaxBytesReader(w, r.Body, config.MaxRequestBodySize)

			next.ServeHTTP(w, r)
		})
	}
}

// CORSMiddleware handles Cross-Origin Resource Sharing.
func CORSMiddleware(config SecurityConfig, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			w.Header().Add("Vary", "Origin")

			// Check if origin is allowed
			allowed := false
			for _, o := range config.CORSAllowedOrigins {
				if origin != "" && (o == origin || (o == "*" && !config.CORSAllowCredentials)) {
					allowed = true
					break
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				if config.CORSAllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.CORSAllowedMethods, ", "))
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.CORSAllowedHeaders, ", "))
				w.Header().Set("Access-Control-Max-Age", "86400")
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeadersMiddleware adds security-related HTTP headers.
func SecurityHeadersMiddleware(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Prevent MIME type sniffing
			w.Header().Set("X-Content-Type-Options", "nosniff")

			// Prevent clickjacking
			w.Header().Set("X-Frame-Options", "DENY")

			// XSS protection (legacy but still useful)
			w.Header().Set("X-XSS-Protection", "1; mode=block")

			// Referrer policy
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Content Security Policy - restrictive but allows our SPA
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self' https:; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")

			// Permissions policy
			w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")

			next.ServeHTTP(w, r)
		})
	}
}

// TimeoutMiddleware adds request timeout.
func TimeoutMiddleware(config SecurityConfig, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), config.RequestTimeout)
			defer cancel()

			r = r.WithContext(ctx)

			// Keep ResponseWriter ownership on the serving goroutine. Database and
			// outbound calls observe ctx; the server enforces socket deadlines.
			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the client IP from the request.
func getClientIP(r *http.Request) string {
	// Forwarding headers are attacker controlled unless a trusted proxy has
	// been configured. Limit by the direct peer by default.
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// isPrivateIP checks if an IP is in private ranges.
// ValidateOutboundHost blocks hostnames that resolve only to private/loopback/link-local
// addresses (SSRF protection for non-HTTP outbound connections such as IMAP).
func ValidateOutboundHost(ctx context.Context, host string, blockPrivate bool, allowedCIDRs []string) error {
	if strings.TrimSpace(host) == "" {
		return errors.New("host required")
	}
	// Literal IP: check directly, no DNS.
	if ip := net.ParseIP(host); ip != nil {
		if blockPrivate && isPrivateIP(ip.String()) {
			if !cidrAllowed(ip, allowedCIDRs) {
				return errors.New("private/loopback IP addresses are blocked")
			}
		}
		return nil
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return fmt.Errorf("resolve host: %w", err)
	}
	if len(ips) == 0 {
		return errors.New("hostname resolved to no addresses")
	}
	for _, a := range ips {
		if blockPrivate && isPrivateIP(a.IP.String()) && !cidrAllowed(a.IP, allowedCIDRs) {
			return fmt.Errorf("host %s resolves to blocked address %s", host, a.IP)
		}
	}
	return nil
}

func cidrAllowed(ip net.IP, cidrs []string) bool {
	for _, c := range cidrs {
		if _, n, err := net.ParseCIDR(c); err == nil && n.Contains(ip) {
			return true
		}
	}
	return false
}

func isPrivateIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return true
	}
	if !parsedIP.IsGlobalUnicast() || parsedIP.IsPrivate() || parsedIP.IsLoopback() || parsedIP.IsLinkLocalUnicast() {
		return true
	}

	// Check private ranges
	privateRanges := []string{
		"0.0.0.0/8",
		"192.0.0.0/24",
		"198.18.0.0/15",
		"240.0.0.0/4",
		"64:ff9b::/96",
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"100.64.0.0/10",
		"::1/128",
		"fc00::/7",
		"fe80::/10",
	}

	for _, cidr := range privateRanges {
		_, network, _ := net.ParseCIDR(cidr)
		if network.Contains(parsedIP) {
			return true
		}
	}

	return false
}

// SSRFProtectionMiddleware protects against Server-Side Request Forgery.
func SSRFProtectionMiddleware(config SecurityConfig, log *slog.Logger) func(http.Handler) http.Handler {
	if !config.SSRFBlockPrivateIPs {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// This middleware should wrap HTTP client calls that make outbound requests
			// For now, we just log - the actual protection is in the HTTP client used for webhooks/integrations
			next.ServeHTTP(w, r)
		})
	}
}

// ValidateSSRFURL validates a URL for SSRF protection.
// Returns error if URL points to private/internal addresses.
func ValidateSSRFURL(rawURL string, config SecurityConfig) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return err
	}

	// Block non-HTTP(S) schemes
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.New("only http and https schemes allowed")
	}

	host := parsedURL.Hostname()
	if host == "" || parsedURL.User != nil {
		return errors.New("URL must have a host and no user credentials")
	}

	// Resolve hostname
	ips, err := net.LookupIP(host)
	if err != nil {
		return errors.New("failed to resolve hostname")
	}

	// Parse allowed networks from config
	allowedNetworks := make([]*net.IPNet, 0)
	for _, cidr := range config.SSRFAllowedCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil {
			allowedNetworks = append(allowedNetworks, network)
		}
	}

	for _, ip := range ips {
		if isPrivateIP(ip.String()) {
			// Check if explicitly allowed
			allowed := false
			for _, network := range allowedNetworks {
				if network.Contains(ip) {
					allowed = true
					break
				}
			}
			if !allowed {
				return errors.New("access to private IP addresses is blocked")
			}
		}
	}

	return nil
}

// CrossOriginMiddleware protects cookie-authenticated writes, including login
// and logout, using browser Origin and Fetch Metadata headers.
func CrossOriginMiddleware(origins []string) func(http.Handler) http.Handler {
	protection := http.NewCrossOriginProtection()
	for _, origin := range origins {
		if origin != "*" {
			_ = protection.AddTrustedOrigin(strings.TrimSpace(origin))
		}
	}
	return protection.Handler
}

// CSRFConfig holds double-submit cookie CSRF settings.
type CSRFConfig struct {
	Enabled      bool
	HeaderName   string   // expected request header carrying the token
	CookieName   string   // cookie holding the token (readable by the SPA)
	TokenBytes   int      // random bytes behind each token
	CookieSecure bool
	// ExemptPaths are exact paths that may issue state-changing requests
	// before any CSRF cookie exists (e.g. the initial login). They must be
	// covered by other protections such as cross-origin checks.
	ExemptPaths []string
}

func (c CSRFConfig) withDefaults() CSRFConfig {
	if c.HeaderName == "" {
		c.HeaderName = "X-CSRF-Token"
	}
	if c.CookieName == "" {
		c.CookieName = "csrf_token"
	}
	if c.TokenBytes <= 0 {
		c.TokenBytes = 32
	}
	return c
}

// CSRFMiddleware implements the double-submit cookie pattern: every response
// carries a CSRF cookie (readable by JavaScript, not HttpOnly), and every
// state-changing request must echo it in the configured header. Requests
// authenticated with a Bearer token instead of cookies (MCP API keys) are
// exempt because they cannot be forged cross-site.
func CSRFMiddleware(config CSRFConfig) func(http.Handler) http.Handler {
	config = config.withDefaults()
	if !config.Enabled {
		return func(next http.Handler) http.Handler { return next }
	}
	safe := func(method string) bool {
		switch method {
		case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
			return true
		}
		return false
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := ""
			if existing, err := r.Cookie(config.CookieName); err == nil {
				token = existing.Value
			}
			if token == "" {
				var err error
				token, err = newCSRFToken(config.TokenBytes)
				if err != nil {
					http.Error(w, "csrf token unavailable", http.StatusInternalServerError)
					return
				}
				http.SetCookie(w, &http.Cookie{
					Name:     config.CookieName,
					Value:    token,
					Path:     "/",
					SameSite: http.SameSiteLaxMode,
					Secure:   config.CookieSecure || r.TLS != nil,
				})
			}
			if safe(r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			// Bearer-authenticated clients (MCP) are not cookie sessions.
			if strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
				next.ServeHTTP(w, r)
				return
			}
			// Paths that must work before any CSRF cookie exists (initial login).
			for _, p := range config.ExemptPaths {
				if r.URL.Path == p {
					next.ServeHTTP(w, r)
					return
				}
			}
			header := r.Header.Get(config.HeaderName)
			cookie, err := r.Cookie(config.CookieName)
			if err != nil || cookie.Value == "" || header == "" ||
				subtle.ConstantTimeCompare([]byte(header), []byte(cookie.Value)) != 1 {
				http.Error(w, "csrf validation failed", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func newCSRFToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
