package email

import (
	"fmt"
	"net/mail"
	"os"
	"strconv"
	"strings"
)

func SMTPConfigFromEnvironment() (*SMTPConfig, error) {
	host := strings.TrimSpace(os.Getenv("CRM_SMTP_HOST"))
	if host == "" {
		return nil, nil
	}
	if strings.ContainsAny(host, "\r\n /\\") {
		return nil, fmt.Errorf("invalid CRM_SMTP_HOST")
	}
	port := 587
	if raw := os.Getenv("CRM_SMTP_PORT"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 65535 {
			return nil, fmt.Errorf("invalid CRM_SMTP_PORT")
		}
		port = value
	}
	config := &SMTPConfig{Host: host, Port: port, Username: os.Getenv("CRM_SMTP_USERNAME"), Password: os.Getenv("CRM_SMTP_PASSWORD"), From: os.Getenv("CRM_SMTP_FROM"), UseTLS: port == 465}
	if raw := os.Getenv("CRM_SMTP_IMPLICIT_TLS"); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid CRM_SMTP_IMPLICIT_TLS")
		}
		config.UseTLS = value
	}
	if config.From == "" {
		config.From = config.Username
	}
	if _, err := mail.ParseAddress(config.From); err != nil {
		return nil, fmt.Errorf("CRM_SMTP_FROM must be a valid email address")
	}
	return config, nil
}
