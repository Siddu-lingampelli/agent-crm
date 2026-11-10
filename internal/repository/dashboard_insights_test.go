package repository

import (
	"context"
	"crm.local/internal/db"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func TestDashboardInsightsAndActionableQueues(t *testing.T) {
	ctx := context.Background()
	database, err := db.Open(ctx, db.Config{Path: filepath.Join(t.TempDir(), "dashboard.db"), MaxOpenConns: 2, MaxIdleConns: 2}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := database.SQL().ExecContext(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	for _, ws := range []string{"dash-one", "dash-two"} {
		exec("INSERT INTO workspaces(id,name,created_at) VALUES(?,?,?)", ws, ws, now)
		exec("INSERT INTO pipelines(id,workspace_id,name,is_default,created_at,updated_at) VALUES(?,?,?,1,?,?)", ws+"-pipeline", ws, "Sales", now, now)
	}
	for i, category := range []string{"open", "won", "lost", "open"} {
		exec("INSERT INTO pipeline_stages(id,pipeline_id,name,position,category,created_at) VALUES(?,'dash-one-pipeline',?,?,?,?)", fmt.Sprint("dash-stage-", i), category, i, category, now)
	}
	exec("INSERT INTO pipeline_stages(id,pipeline_id,name,position,category,created_at) VALUES('dash-foreign-stage','dash-two-pipeline','open',0,'open',?)", now)
	insertDeal := func(id, ws, stage, currency, close string, amount any, probability int, won any, archived any) {
		t.Helper()
		pipeline := ws + "-pipeline"
		exec(`INSERT INTO deals(id,workspace_id,name,pipeline_id,stage_id,amount,currency,probability,expected_close,source,lost_reason,created_by,updated_by,created_at,updated_at,won_at,archived_at) VALUES(?,?,?,?,?,?,?,?,?,'','','','',?,?,?,?)`, id, ws, id, pipeline, stage, amount, currency, probability, close, "2026-09-01 10:00:00 +0000 UTC", "2026-10-01 10:00:00.123 +0000 UTC", won, archived)
	}
	insertDeal("open-usd", "dash-one", "dash-stage-0", "USD", "2026-11-20", 10000, 50, nil, nil)
	insertDeal("open-eur", "dash-one", "dash-stage-0", "EUR", "2026-11-09", 20000, 25, nil, nil)
	insertDeal("open-unvalued", "dash-one", "dash-stage-0", "USD", "", nil, 50, nil, nil)
	insertDeal("won-usd-now", "dash-one", "dash-stage-1", "USD", "2026-11-10", 6000, 100, "2026-11-10 10:00:00 +0000 UTC", nil)
	insertDeal("won-usd-offset", "dash-one", "dash-stage-1", "USD", "2026-10-31", 4000, 100, "2026-11-01T00:15:00+05:30", nil)
	insertDeal("won-eur", "dash-one", "dash-stage-1", "EUR", "2026-11-05", 8000, 100, "2026-11-05T10:00:00Z", nil)
	insertDeal("lost-usd", "dash-one", "dash-stage-2", "USD", "2026-11-01", 5000, 0, nil, nil)
	insertDeal("archived-usd", "dash-one", "dash-stage-0", "USD", "2026-11-01", 999999, 100, nil, now)
	insertDeal("foreign-usd", "dash-two", "dash-foreign-stage", "USD", "2026-11-01", 999999, 100, nil, nil)
	for i := 0; i < 300; i++ {
		insertDeal(fmt.Sprint("bulk-deal-", i), "dash-one", "dash-stage-0", "USD", "2026-11-20", 100, 50, nil, nil)
	}
	for i := 0; i < 300; i++ {
		exec("INSERT INTO tasks(id,workspace_id,title,status,due_date,created_by,created_at,updated_at) VALUES(?,'dash-one','Daily follow-up','open','2026-11-10','',?,?)", fmt.Sprint("bulk-task-", i), now, now)
	}
	for _, row := range []struct {
		id, status string
		due        any
	}{{"past", "open", "2026-11-09"}, {"future", "in_progress", "2026-11-11"}, {"undated", "open", nil}, {"complete", "completed", "2026-11-09"}, {"cancelled", "cancelled", "2026-11-10"}, {"done", "done", "2026-11-10"}} {
		exec("INSERT INTO tasks(id,workspace_id,title,status,due_date,created_by,created_at,updated_at) VALUES(?,'dash-one',?,?,?,'',?,?)", row.id, row.id, row.status, row.due, now, now)
	}
	exec("INSERT INTO tasks(id,workspace_id,title,status,due_date,created_by,created_at,updated_at) VALUES('foreign-task','dash-two','Foreign','open','2026-11-10','',?,?)", now, now)
	exec("INSERT INTO leads(id,workspace_id,display_name,status,created_at,updated_at) VALUES('legacy-new-lead','dash-one','New','new','2026-11-10 10:00:00 +0000 UTC',?)", now)
	repo := NewDealRepository(database.SQL())
	insights, err := repo.DashboardInsights(ctx, "dash-one", now, now)
	if err != nil {
		t.Fatal(err)
	}
	if insights.OpenDeals != 303 || insights.PastCloseDeals != 1 || insights.InactiveDeals != 303 || insights.WonDeals != 3 || insights.LostDeals != 1 || insights.ActiveTasks != 303 || insights.UnscheduledTasks != 1 {
		t.Fatalf("incorrect full workspace totals: %+v", insights)
	}
	if insights.PipelineID != "dash-one-pipeline" || len(insights.Stages) != 6 {
		t.Fatalf("pipeline scope/empty stages incorrect: %+v", insights.Stages)
	}
	for _, currency := range insights.Currencies {
		switch currency.Currency {
		case "USD":
			if currency.OpenValue != 40000 || currency.WeightedValue != 20000 || currency.WonThisMonth != 6000 || currency.WonLastMonth != 4000 || currency.AverageWon != 5000 || currency.OpenDeals != 302 {
				t.Fatalf("USD mixed or truncated: %+v", currency)
			}
		case "EUR":
			if currency.OpenValue != 20000 || currency.WeightedValue != 5000 || currency.WonThisMonth != 8000 {
				t.Fatalf("EUR mixed: %+v", currency)
			}
		default:
			t.Fatalf("unexpected currency: %+v", currency)
		}
	}
	if len(insights.Currencies) != 2 || len(insights.Revenue) != 3 {
		t.Fatalf("currency/revenue buckets incorrect: %+v", insights)
	}
	for _, point := range insights.Revenue {
		if point.Currency == "USD" && point.Month == "2026-10" && point.Value != 4000 {
			t.Fatalf("UTC revenue boundary failed: %+v", point)
		}
	}
	totals, err := repo.DashboardTotals(ctx, "dash-one", now)
	if err != nil {
		t.Fatal(err)
	}
	if totals.TasksDue != 300 || totals.TasksOverdue != 1 || totals.NewLeadsThisWeek != 1 {
		t.Fatalf("daily/legacy lead totals incorrect: %+v", totals)
	}
	taskRepo := NewTaskRepository(database.SQL())
	for _, test := range []struct {
		opts  TaskListOptions
		total int
		id    string
	}{{TaskListOptions{ActiveOnly: true, DueOn: "2026-11-10"}, 300, ""}, {TaskListOptions{ActiveOnly: true, DueBefore: "2026-11-10"}, 1, "past"}, {TaskListOptions{ActiveOnly: true, DueAfter: "2026-11-10"}, 1, "future"}, {TaskListOptions{ActiveOnly: true, Unscheduled: true}, 1, "undated"}} {
		test.opts.WorkspaceID = "dash-one"
		test.opts.Limit = 5
		page, err := taskRepo.List(ctx, test.opts)
		if err != nil {
			t.Fatal(err)
		}
		if page.Total != test.total || len(page.Items) > 5 || (test.id != "" && page.Items[0].ID != test.id) {
			t.Fatalf("task queue truncated or crossed scope: %+v", page)
		}
	}
	for _, test := range []struct {
		opts  DealListOptions
		total int
	}{{DealListOptions{Category: "open", CloseFrom: "2026-11-10", CloseBefore: "2026-12-11"}, 301}, {DealListOptions{Category: "open", CloseBefore: "2026-11-10"}, 1}, {DealListOptions{Category: "open", InactiveBefore: "2026-10-27T12:00:00Z"}, 303}, {DealListOptions{Category: "' OR 1=1 --"}, 0}} {
		test.opts.WorkspaceID = "dash-one"
		test.opts.Limit = 6
		page, err := repo.List(ctx, test.opts)
		if err != nil {
			t.Fatal(err)
		}
		if page.Total != test.total || len(page.Items) > 6 {
			t.Fatalf("deal queue scope/filter failed: %+v", page)
		}
	}
	defaults, err := repo.ListDefaultPipeline(ctx, "dash-one")
	if err != nil || len(defaults) != 1 {
		t.Fatalf("default pipeline query failed: %+v %v", defaults, err)
	}
	empty, err := repo.DashboardInsights(ctx, "empty", now, now)
	if err != nil || empty.OpenDeals != 0 || len(empty.Currencies) != 0 || len(empty.Stages) != 0 {
		t.Fatalf("empty workspace failed: %+v %v", empty, err)
	}
}
