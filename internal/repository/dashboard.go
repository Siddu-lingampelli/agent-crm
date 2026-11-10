package repository

import (
	"context"
	"time"
)

type DashboardTotals struct {
	OpenValue, WeightedValue, AvgDealSize                              int64
	NewLeads, NewLeadsThisWeek, TasksDue, TasksOverdue, SalesCycleDays int
	WinRate, LeadConversionRate                                        float64
}

// Aggregate in SQLite rather than materializing a capped page of records.
func (r *DealRepository) DashboardTotals(ctx context.Context, workspace string, now time.Time) (DashboardTotals, error) {
	var out DashboardTotals
	var won, lost int
	err := r.db.QueryRowContext(ctx, `SELECT
 COALESCE(SUM(CASE WHEN s.category='open' THEN d.amount ELSE 0 END),0),
 COALESCE(SUM(CASE WHEN s.category='open' THEN d.amount*d.probability/100 ELSE 0 END),0),
 COALESCE(SUM(CASE WHEN s.category='won' THEN 1 ELSE 0 END),0),
 COALESCE(SUM(CASE WHEN s.category='lost' THEN 1 ELSE 0 END),0),
 CAST(COALESCE(AVG(CASE WHEN s.category='won' THEN d.amount END),0) AS INTEGER),
 CAST(COALESCE(AVG(CASE WHEN s.category='won' AND d.won_at IS NOT NULL THEN MAX(0,`+sqliteInstant("d.won_at")+`-`+sqliteInstant("d.created_at")+`) END),0) AS INTEGER)
 FROM deals d JOIN pipeline_stages s ON s.id=d.stage_id WHERE d.workspace_id=? AND d.archived_at IS NULL`, workspace).Scan(&out.OpenValue, &out.WeightedValue, &won, &lost, &out.AvgDealSize, &out.SalesCycleDays)
	if err != nil {
		return out, err
	}
	if won+lost > 0 {
		out.WinRate = float64(won) / float64(won+lost) * 100
	}
	weekday := (int(now.Weekday()) + 6) % 7
	weekStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -weekday)
	var total, converted int
	err = r.db.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(SUM(CASE WHEN status='new' THEN 1 ELSE 0 END),0),COALESCE(SUM(CASE WHEN `+sqliteInstant("created_at")+`>=julianday(?) THEN 1 ELSE 0 END),0),COALESCE(SUM(CASE WHEN status='converted' THEN 1 ELSE 0 END),0) FROM leads WHERE workspace_id=? AND archived_at IS NULL`, weekStart.Format(time.RFC3339), workspace).Scan(&total, &out.NewLeads, &out.NewLeadsThisWeek, &converted)
	if err != nil {
		return out, err
	}
	if total > 0 {
		out.LeadConversionRate = float64(converted) / float64(total) * 100
	}
	err = r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(CASE WHEN substr(due_date,1,10)=? THEN 1 ELSE 0 END),0),COALESCE(SUM(CASE WHEN due_date!='' AND substr(due_date,1,10)<? THEN 1 ELSE 0 END),0) FROM tasks WHERE workspace_id=? AND archived_at IS NULL AND status NOT IN ('done','completed','cancelled')`, now.Format("2006-01-02"), now.Format("2006-01-02"), workspace).Scan(&out.TasksDue, &out.TasksOverdue)
	return out, err
}
