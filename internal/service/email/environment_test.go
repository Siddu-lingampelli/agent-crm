package email

import "testing"

func TestSMTPEnvironment(t *testing.T) {
	t.Setenv("CRM_SMTP_HOST", "smtp.example.com")
	t.Setenv("CRM_SMTP_PORT", "465")
	t.Setenv("CRM_SMTP_FROM", "owner@example.com")
	t.Setenv("CRM_SMTP_USERNAME", "owner@example.com")
	t.Setenv("CRM_SMTP_PASSWORD", "test-secret")
	t.Setenv("CRM_SMTP_IMPLICIT_TLS", "")
	config, err := SMTPConfigFromEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	if !config.UseTLS || config.Port != 465 || config.Password != "test-secret" {
		t.Fatal("SMTP configuration ignored")
	}
	t.Setenv("CRM_SMTP_PORT", "invalid")
	if _, err := SMTPConfigFromEnvironment(); err == nil {
		t.Fatal("invalid SMTP port accepted")
	}
}
