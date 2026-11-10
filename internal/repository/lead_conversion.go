package repository

import (
	"context"
	"crm.local/internal/model"
	"crm.local/internal/uuid"
	"strings"
	"time"
)

type LeadConversion struct {
	ContactID, CompanyID, DealName, PipelineID, StageID, Currency, ExpectedClose string
	Amount                                                                       int64
}

// ConvertWithRelations creates related records and marks the lead converted in
// one transaction. Validation failures cannot leave partial contacts or deals.
func (r *LeadRepository) ConvertWithRelations(ctx context.Context, workspace, actor, id string, in LeadConversion) (*model.Lead, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	result, err := tx.ExecContext(ctx, "UPDATE leads SET status='converted',converted_at=?,updated_at=?,updated_by=? WHERE id=? AND workspace_id=? AND archived_at IS NULL AND status<>'converted'", now, now, actor, id, workspace)
	if err != nil {
		return nil, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if count != 1 {
		return nil, invalid("lead is unavailable or already converted")
	}
	lead, err := scanLead(tx.QueryRowContext(ctx, "SELECT "+leadsColumns+" FROM leads WHERE id=? AND workspace_id=?", id, workspace))
	if err != nil {
		return nil, err
	}
	companyID := in.CompanyID
	if companyID == "" {
		companyID = lead.CompanyID
	}
	if companyID == "" && strings.TrimSpace(lead.CompanyName) != "" {
		companyID = uuid.New()
		if _, err := tx.ExecContext(ctx, "INSERT INTO companies(id,workspace_id,name,created_at,updated_at) VALUES(?,?,?,?,?)", companyID, workspace, lead.CompanyName, now, now); err != nil {
			return nil, err
		}
	}
	if err := validateEntityReferences(ctx, tx, &model.Lead{DisplayName: lead.DisplayName, WorkspaceID: workspace, CompanyID: companyID}); err != nil {
		return nil, err
	}
	contactID := in.ContactID
	if contactID == "" {
		contactID = uuid.New()
		first := lead.FirstName
		if strings.TrimSpace(first) == "" {
			first = lead.DisplayName
		}
		var company any
		if companyID != "" {
			company = companyID
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO contacts(id,workspace_id,first_name,last_name,email,phone,job_title,company_id,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?)", contactID, workspace, first, lead.LastName, lead.Email, lead.Phone, lead.JobTitle, company, now, now); err != nil {
			return nil, err
		}
	} else {
		var count int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM contacts WHERE id=? AND workspace_id=? AND archived_at IS NULL", contactID, workspace).Scan(&count); err != nil {
			return nil, err
		}
		if count != 1 {
			return nil, ErrInvalidReference
		}
	}
	if strings.TrimSpace(in.DealName) != "" {
		pipelineID := in.PipelineID
		if pipelineID == "" {
			if err := tx.QueryRowContext(ctx, "SELECT id FROM pipelines WHERE workspace_id=? ORDER BY is_default DESC,id LIMIT 1", workspace).Scan(&pipelineID); err != nil {
				return nil, invalid("a workspace pipeline is required")
			}
		}
		stageID := in.StageID
		if stageID == "" {
			if err := tx.QueryRowContext(ctx, "SELECT id FROM pipeline_stages WHERE pipeline_id=? AND category='open' ORDER BY position,id LIMIT 1", pipelineID).Scan(&stageID); err != nil {
				return nil, invalid("an open pipeline stage is required")
			}
		}
		currency := in.Currency
		if currency == "" {
			currency = lead.Currency
		}
		if currency == "" {
			currency = "USD"
		}
		deal := &model.Deal{ID: uuid.New(), WorkspaceID: workspace, Name: in.DealName, PipelineID: pipelineID, StageID: stageID, CompanyID: companyID, PrimaryContactID: contactID, LeadID: id, Amount: &in.Amount, Currency: currency, ExpectedClose: in.ExpectedClose}
		if err := validateEntityReferences(ctx, tx, deal); err != nil {
			return nil, err
		}
		var probability int
		var category string
		if err := tx.QueryRowContext(ctx, "SELECT probability,category FROM pipeline_stages WHERE id=?", stageID).Scan(&probability, &category); err != nil {
			return nil, err
		}
		var company, wonAt, lostAt any
		if companyID != "" {
			company = companyID
		}
		if category == "won" {
			wonAt = now
		}
		if category == "lost" {
			lostAt = now
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO deals(id,workspace_id,name,pipeline_id,stage_id,amount,currency,probability,expected_close,company_id,primary_contact_id,lead_id,created_by,updated_by,created_at,updated_at,won_at,lost_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)", deal.ID, workspace, deal.Name, pipelineID, stageID, in.Amount, currency, probability, in.ExpectedClose, company, contactID, id, actor, actor, now, now, wonAt, lostAt); err != nil {
			return nil, err
		}
	}
	var company any
	if companyID != "" {
		company = companyID
	}
	if _, err := tx.ExecContext(ctx, "UPDATE leads SET company_id=? WHERE id=? AND workspace_id=?", company, id, workspace); err != nil {
		return nil, err
	}
	lead.CompanyID = companyID
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return lead, nil
}
