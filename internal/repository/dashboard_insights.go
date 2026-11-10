package repository

import (
	"context"
	"database/sql"
	"time"
)

// sqliteInstant also accepts UTC timestamps written by the legacy Go driver.
// Callers supply fixed column expressions, never request parameters.
func sqliteInstant(column string) string {
	return "COALESCE(julianday(" + column + "), julianday(replace(replace(" + column + ", ' +0000 UTC', 'Z'), ' ', 'T')))"
}

type DashboardInsights struct {
	Day              string              `json:"day"`
	PipelineID       string              `json:"pipeline_id"`
	PipelineName     string              `json:"pipeline_name"`
	OpenDeals        int                 `json:"open_deals"`
	WonDeals         int                 `json:"won_deals"`
	LostDeals        int                 `json:"lost_deals"`
	PastCloseDeals   int                 `json:"past_close_deals"`
	InactiveDeals    int                 `json:"inactive_deals"`
	TotalLeads       int                 `json:"total_leads"`
	ConvertedLeads   int                 `json:"converted_leads"`
	Companies        int                 `json:"companies"`
	Contacts         int                 `json:"contacts"`
	ActiveTasks      int                 `json:"active_tasks"`
	UnscheduledTasks int                 `json:"unscheduled_tasks"`
	Currencies       []DashboardCurrency `json:"currencies"`
	Stages           []DashboardStage    `json:"stages"`
	Revenue          []DashboardRevenue  `json:"revenue"`
}
type DashboardCurrency struct {
	Currency      string `json:"currency"`
	OpenValue     int64  `json:"open_value"`
	WeightedValue int64  `json:"weighted_value"`
	WonThisMonth  int64  `json:"won_this_month"`
	WonLastMonth  int64  `json:"won_last_month"`
	AverageWon    int64  `json:"average_won"`
	OpenDeals     int    `json:"open_deals"`
	WonDeals      int    `json:"won_deals"`
	LostDeals     int    `json:"lost_deals"`
}
type DashboardStage struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Position int    `json:"position"`
	Currency string `json:"currency"`
	Count    int    `json:"count"`
	Value    int64  `json:"value"`
}
type DashboardRevenue struct {
	Month    string `json:"month"`
	Currency string `json:"currency"`
	Value    int64  `json:"value"`
	Count    int    `json:"count"`
}

// DashboardInsights aggregates complete workspace records. Monetary totals are
// grouped by currency; no exchange rates or synthetic revenue are applied.
func (r *DealRepository) DashboardInsights(ctx context.Context, workspace string, day, now time.Time) (DashboardInsights, error) {
	out := DashboardInsights{Day: day.Format("2006-01-02"), Currencies: []DashboardCurrency{}, Stages: []DashboardStage{}, Revenue: []DashboardRevenue{}}
	currency := "COALESCE(NULLIF(UPPER(TRIM(d.currency)),''),'USD')"
	instant := sqliteInstant("d.won_at")
	updated := sqliteInstant("d.updated_at")
	month := time.Date(day.Year(), day.Month(), 1, 0, 0, 0, 0, time.UTC)
	previous := month.AddDate(0, -1, 0).Format(time.RFC3339)
	start := month.Format(time.RFC3339)
	end := month.AddDate(0, 1, 0).Format(time.RFC3339)
	err := r.db.QueryRowContext(ctx, `SELECT
 COALESCE(SUM(s.category='open'),0),COALESCE(SUM(s.category='won'),0),COALESCE(SUM(s.category='lost'),0),
 COALESCE(SUM(s.category='open' AND d.expected_close!='' AND substr(d.expected_close,1,10)<?),0),
 COALESCE(SUM(s.category='open' AND `+updated+`<julianday(?)),0)
 FROM deals d JOIN pipeline_stages s ON s.id=d.stage_id WHERE d.workspace_id=? AND d.archived_at IS NULL`, out.Day, now.AddDate(0, 0, -14).Format(time.RFC3339Nano), workspace).Scan(&out.OpenDeals, &out.WonDeals, &out.LostDeals, &out.PastCloseDeals, &out.InactiveDeals)
	if err != nil {
		return out, err
	}
	err = r.db.QueryRowContext(ctx, `SELECT
 (SELECT COUNT(*) FROM leads WHERE workspace_id=? AND archived_at IS NULL),
 (SELECT COUNT(*) FROM leads WHERE workspace_id=? AND archived_at IS NULL AND status='converted'),
 (SELECT COUNT(*) FROM companies WHERE workspace_id=? AND archived_at IS NULL),
 (SELECT COUNT(*) FROM contacts WHERE workspace_id=? AND archived_at IS NULL),
 (SELECT COUNT(*) FROM tasks WHERE workspace_id=? AND archived_at IS NULL AND status NOT IN ('done','completed','cancelled')),
 (SELECT COUNT(*) FROM tasks WHERE workspace_id=? AND archived_at IS NULL AND status NOT IN ('done','completed','cancelled') AND (due_date IS NULL OR due_date=''))`, workspace, workspace, workspace, workspace, workspace, workspace).Scan(&out.TotalLeads, &out.ConvertedLeads, &out.Companies, &out.Contacts, &out.ActiveTasks, &out.UnscheduledTasks)
	if err != nil {
		return out, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT `+currency+`,
 COALESCE(SUM(CASE WHEN s.category='open' THEN d.amount END),0),
 CAST(COALESCE(SUM(CASE WHEN s.category='open' THEN d.amount*MAX(0,MIN(100,d.probability))/100.0 END),0) AS INTEGER),
 COALESCE(SUM(CASE WHEN s.category='won' AND `+instant+`>=julianday(?) AND `+instant+`<julianday(?) THEN d.amount END),0),
 COALESCE(SUM(CASE WHEN s.category='won' AND `+instant+`>=julianday(?) AND `+instant+`<julianday(?) THEN d.amount END),0),
 CAST(COALESCE(AVG(CASE WHEN s.category='won' THEN d.amount END),0) AS INTEGER),
 SUM(s.category='open'),SUM(s.category='won'),SUM(s.category='lost')
 FROM deals d JOIN pipeline_stages s ON s.id=d.stage_id WHERE d.workspace_id=? AND d.archived_at IS NULL GROUP BY `+currency+` ORDER BY 1`, start, end, previous, start, workspace)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item DashboardCurrency
		if err = rows.Scan(&item.Currency, &item.OpenValue, &item.WeightedValue, &item.WonThisMonth, &item.WonLastMonth, &item.AverageWon, &item.OpenDeals, &item.WonDeals, &item.LostDeals); err != nil {
			rows.Close()
			return out, err
		}
		out.Currencies = append(out.Currencies, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	// The stage distribution has an explicit pipeline scope and includes empty stages.
	err = r.db.QueryRowContext(ctx, `SELECT id,name FROM pipelines WHERE workspace_id=? ORDER BY is_default DESC,created_at ASC,id ASC LIMIT 1`, workspace).Scan(&out.PipelineID, &out.PipelineName)
	if err != nil && err != sql.ErrNoRows {
		return out, err
	}
	if out.PipelineID != "" {
		rows, err = r.db.QueryContext(ctx, `SELECT s.id,s.name,s.category,s.position,CASE WHEN d.id IS NULL THEN '' ELSE `+currency+` END,COUNT(d.id),COALESCE(SUM(d.amount),0)
 FROM pipeline_stages s LEFT JOIN deals d ON d.stage_id=s.id AND d.workspace_id=? AND d.archived_at IS NULL
 WHERE s.pipeline_id=? GROUP BY s.id,`+currency+` ORDER BY s.position,s.id`, workspace, out.PipelineID)
		if err != nil {
			return out, err
		}
		for rows.Next() {
			var item DashboardStage
			if err = rows.Scan(&item.ID, &item.Name, &item.Category, &item.Position, &item.Currency, &item.Count, &item.Value); err != nil {
				rows.Close()
				return out, err
			}
			out.Stages = append(out.Stages, item)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return out, err
		}
	}
	rows, err = r.db.QueryContext(ctx, `SELECT strftime('%Y-%m',`+instant+`),`+currency+`,COALESCE(SUM(d.amount),0),COUNT(*)
 FROM deals d JOIN pipeline_stages s ON s.id=d.stage_id WHERE d.workspace_id=? AND d.archived_at IS NULL AND s.category='won' AND `+instant+`>=julianday(?) AND `+instant+`<julianday(?) GROUP BY 1,2 ORDER BY 1,2`, workspace, month.AddDate(0, -5, 0).Format(time.RFC3339), end)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var item DashboardRevenue
		if err = rows.Scan(&item.Month, &item.Currency, &item.Value, &item.Count); err != nil {
			return out, err
		}
		out.Revenue = append(out.Revenue, item)
	}
	return out, rows.Err()
}
