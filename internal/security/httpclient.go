// Package security implements security middleware for the HTTP server.
package security

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"time"
)

// SecureHTTPClientConfig holds configuration for the secure HTTP client.
type SecureHTTPClientConfig struct {
	// Timeout for the entire request
	Timeout time.Duration

	// TLS configuration
	InsecureSkipVerify bool // Should be false in production

	// SSRF protection
	BlockPrivateIPs bool
	AllowedCIDRs    []string

	// Limits
	MaxResponseSize int64 // bytes, 0 = no limit
}

// DefaultSecureHTTPClientConfig returns a production-ready config.
func DefaultSecureHTTPClientConfig() SecureHTTPClientConfig {
	return SecureHTTPClientConfig{
		Timeout:            30 * time.Second,
		InsecureSkipVerify: false,
		BlockPrivateIPs:    true,
		MaxResponseSize:    10 * 1024 * 1024, // 10 MB
	}
}

// SecureHTTPClient returns an HTTP client with SSRF protection.
func SecureHTTPClient(config SecureHTTPClientConfig) *http.Client {
	// Create a custom dialer
	dialer := &net.Dialer{
		Timeout:   config.Timeout,
		KeepAlive: 30 * time.Second,
	}

	// Create a custom transport with SSRF validation
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, err
			}
			if len(ips) == 0 {
				return nil, errors.New("hostname resolved to no addresses")
			}
			for _, address := range ips {
				if config.BlockPrivateIPs && isPrivateIP(address.IP.String()) {
					allowed := false
					for _, cidr := range config.AllowedCIDRs {
						_, block, err := net.ParseCIDR(cidr)
						if err == nil && block.Contains(address.IP) {
							allowed = true
						}
					}
					if !allowed {
						return nil, errors.New("access to private IP addresses is blocked")
					}
				}
			}
			// Dial exactly the validated IP, preventing DNS rebinding between
			// validation and connection. TLS still verifies the original host.
			var dialErr error
			for _, address := range ips {
				conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(address.String(), port))
				if err == nil {
					return conn, nil
				}
				dialErr = err
			}
			return nil, dialErr
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	// Create client with the transport
	client := &http.Client{
		Transport: boundedTransport{base: transport, max: config.MaxResponseSize},
		Timeout:   config.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Validate redirect URLs for SSRF
			if config.BlockPrivateIPs {
				if err := ValidateSSRFURL(req.URL.String(), SecurityConfig{
					SSRFBlockPrivateIPs: true,
					SSRFAllowedCIDRs:    config.AllowedCIDRs,
				}); err != nil {
					return err
				}
			}
			// Allow up to 10 redirects
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	return client
}

type boundedTransport struct {
	base *http.Transport
	max  int64
}

func (t boundedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Scheme != "http" && r.URL.Scheme != "https" {
		return nil, errors.New("only http and https schemes allowed")
	}
	if r.URL.User != nil {
		return nil, errors.New("URL credentials are not allowed")
	}
	response, err := t.base.RoundTrip(r)
	if err != nil {
		return nil, err
	}
	if t.max > 0 {
		if response.ContentLength > t.max {
			_ = response.Body.Close()
			return nil, errors.New("response exceeds maximum size")
		}
		response.Body = &boundedBody{ReadCloser: response.Body, remaining: t.max}
	}
	return response, nil
}
func (t boundedTransport) CloseIdleConnections() { t.base.CloseIdleConnections() }

type boundedBody struct {
	io.ReadCloser
	remaining int64
}

func (b *boundedBody) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if b.remaining <= 0 {
		var extra [1]byte
		n, err := b.ReadCloser.Read(extra[:])
		if n > 0 {
			return 0, errors.New("response exceeds maximum size")
		}
		return 0, err
	}
	if int64(len(p)) > b.remaining {
		p = p[:b.remaining]
	}
	n, err := b.ReadCloser.Read(p)
	b.remaining -= int64(n)
	return n, err
}
