package repository

import (
	"crm.local/internal/model"
	"fmt"
	"net/mail"
	"net/url"
	"strings"
	"time"
)

type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }
func invalid(message string) error       { return &ValidationError{message} }

func validateEntityInput(record any) error {
	var name string
	var dates []string
	switch v := record.(type) {
	case *model.Company:
		name = v.Name
		if v.Website != "" {
			u, err := url.Parse(v.Website)
			if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
				return invalid("website must be an HTTP or HTTPS URL")
			}
		}
	case *model.Contact:
		name = v.FirstName
	case *model.Lead:
		name = v.DisplayName
		if v.LeadScore != nil && (*v.LeadScore < 0 || *v.LeadScore > 100) {
			return invalid("lead_score must be between 0 and 100")
		}
		if v.EstimatedValue != nil && *v.EstimatedValue < 0 {
			return invalid("estimated_value must not be negative")
		}
		dates = []string{v.ExpectedClose, v.FollowUpDate}
	case *model.Deal:
		name = v.Name
		if v.Probability < 0 || v.Probability > 100 {
			return invalid("probability must be between 0 and 100")
		}
		if v.Amount != nil && *v.Amount < 0 {
			return invalid("amount must not be negative")
		}
		dates = []string{v.ExpectedClose}
	case *model.Project:
		name = v.Name
		if v.Progress < 0 || v.Progress > 100 {
			return invalid("progress must be between 0 and 100")
		}
		dates = []string{v.StartDate, v.EndDate}
	case *model.Task:
		name = v.Title
		dates = []string{v.DueDate}
	case *model.CalendarEvent:
		name = v.Title
		start, err := time.Parse(time.RFC3339, v.StartTime)
		if err != nil {
			return invalid("start_time must include a timezone")
		}
		end, err := time.Parse(time.RFC3339, v.EndTime)
		if err != nil || !end.After(start) {
			return invalid("end_time must include a timezone and follow start_time")
		}
	case *model.Email:
		name = v.Subject
		for _, value := range []string{v.To, v.CC, v.BCC, v.Subject} {
			if strings.ContainsAny(value, "\r\n") {
				return invalid("email headers must not contain newlines")
			}
		}
		for _, value := range []string{v.To, v.CC, v.BCC} {
			if value != "" {
				if _, err := mail.ParseAddressList(value); err != nil {
					return invalid("invalid email recipient")
				}
			}
		}
	default:
		return nil
	}
	if strings.TrimSpace(name) == "" {
		return invalid("name or title must not be empty")
	}
	for _, value := range dates {
		if value == "" {
			continue
		}
		if _, err := time.Parse("2006-01-02", value); err != nil {
			if _, err := time.Parse(time.RFC3339, value); err != nil {
				return invalid(fmt.Sprintf("invalid date %q", value))
			}
		}
	}
	return nil
}
