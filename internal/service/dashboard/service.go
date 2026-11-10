// Package dashboard implements dashboard metrics aggregation.
package dashboard

import (
	"context"
	"log/slog"
	"time"

	"crm.local/internal/model"
	"crm.local/internal/permissions"
	"crm.local/internal/repository"
)

// Service aggregates dashboard metrics from various services.
type Service struct {
	dealRepo    *repository.DealRepository
	leadRepo    *repository.LeadRepository
	taskRepo    *repository.TaskRepository
	companyRepo *repository.CompanyRepository
	contactRepo *repository.ContactRepository
	log         *slog.Logger
}

// New returns a dashboard service.
func New(
	dealRepo *repository.DealRepository,
	leadRepo *repository.LeadRepository,
	taskRepo *repository.TaskRepository,
	companyRepo *repository.CompanyRepository,
	contactRepo *repository.ContactRepository,
	log *slog.Logger,
) *Service {
	return &Service{
		dealRepo:    dealRepo,
		leadRepo:    leadRepo,
		taskRepo:    taskRepo,
		companyRepo: companyRepo,
		contactRepo: contactRepo,
		log:         log,
	}
}

// DashboardMetrics contains all dashboard metrics.
type DashboardMetrics struct {
	// Pipeline metrics
	OpenPipelineValue     int64                 `json:"open_pipeline_value"`
	WeightedPipelineValue int64                 `json:"weighted_pipeline_value"`
	PipelineStages        []PipelineStageMetric `json:"pipeline_stages"`

	// Lead metrics
	NewLeadsCount    int `json:"new_leads_count"`
	NewLeadsThisWeek int `json:"new_leads_this_week"`

	// Task metrics
	TasksDueCount     int `json:"tasks_due_count"`
	TasksOverdueCount int `json:"tasks_overdue_count"`

	// Key metrics
	WinRate            float64 `json:"win_rate"`
	AvgDealSize        int64   `json:"avg_deal_size"`
	SalesCycleDays     int     `json:"sales_cycle_days"`
	LeadConversionRate float64 `json:"lead_conversion_rate"`

	// Recent activity
	Insights       repository.DashboardInsights `json:"insights"`
	RecentActivity []ActivityItem               `json:"recent_activity"`
}

// PipelineStageMetric represents a pipeline stage with metrics.
type PipelineStageMetric struct {
	StageID   string `json:"stage_id"`
	StageName string `json:"stage_name"`
	Count     int    `json:"count"`
	Value     int64  `json:"value"`
	Color     string `json:"color"`
	Position  int    `json:"position"`
}

// ActivityItem represents a recent activity.
type ActivityItem struct {
	ID          string `json:"id"`
	UserName    string `json:"user_name"`
	UserAvatar  string `json:"user_avatar"`
	Action      string `json:"action"`
	Target      string `json:"target"`
	TargetURL   string `json:"target_url"`
	Time        string `json:"time"`
	Description string `json:"description"`
}

// GetDashboardMetrics returns aggregated dashboard metrics.
func (s *Service) GetDashboardMetrics(ctx context.Context, actor permissions.AuthContext, scopeDay ...time.Time) (*DashboardMetrics, error) {
	if err := permissions.Authorize(actor, "dashboard:read"); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	workspaceID := actor.WorkspaceID
	day := now
	if len(scopeDay) > 0 {
		day = scopeDay[0]
	}

	totals, err := s.dealRepo.DashboardTotals(ctx, workspaceID, day)
	if err != nil {
		return nil, err
	}
	openPipelineValue := totals.OpenValue
	weightedPipelineValue := totals.WeightedValue
	newLeadsCount := totals.NewLeads
	newLeadsThisWeek := totals.NewLeadsThisWeek
	tasksDueCount := totals.TasksDue
	tasksOverdueCount := totals.TasksOverdue
	winRate := totals.WinRate
	avgDealSize := totals.AvgDealSize
	salesCycleDays := totals.SalesCycleDays
	leadConversionRate := totals.LeadConversionRate
	insights, err := s.dealRepo.DashboardInsights(ctx, workspaceID, day, now)
	if err != nil {
		return nil, err
	}
	pipelineStages := make([]PipelineStageMetric, 0)
	for _, stage := range insights.Stages {
		pipelineStages = append(pipelineStages, PipelineStageMetric{StageID: stage.ID, StageName: stage.Name, Count: stage.Count, Value: stage.Value, Position: stage.Position})
	}
	// Activity is empty until real audit events exist; never invent records.
	recentActivity := []ActivityItem{}

	return &DashboardMetrics{
		OpenPipelineValue:     openPipelineValue,
		WeightedPipelineValue: weightedPipelineValue,
		PipelineStages:        pipelineStages,
		NewLeadsCount:         newLeadsCount,
		NewLeadsThisWeek:      newLeadsThisWeek,
		TasksDueCount:         tasksDueCount,
		TasksOverdueCount:     tasksOverdueCount,
		WinRate:               winRate,
		AvgDealSize:           avgDealSize,
		SalesCycleDays:        salesCycleDays,
		LeadConversionRate:    leadConversionRate,
		Insights:              insights,
		RecentActivity:        recentActivity,
	}, nil
}

func calculateWinRate(deals []model.Deal) float64 {
	var won, lost int
	for _, deal := range deals {
		if deal.WonAt != nil {
			won++
		}
		if deal.LostAt != nil {
			lost++
		}
	}
	total := won + lost
	if total == 0 {
		return 0
	}
	return float64(won) / float64(total) * 100
}

func calculateAvgDealSize(deals []model.Deal) int64 {
	var total int64
	var count int
	for _, deal := range deals {
		if deal.Amount != nil && deal.WonAt != nil {
			total += *deal.Amount
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return total / int64(count)
}

func calculateSalesCycle(deals []model.Deal) int {
	var totalDays int
	var count int
	for _, deal := range deals {
		if deal.WonAt != nil && !deal.CreatedAt.IsZero() {
			days := int(deal.WonAt.Sub(deal.CreatedAt).Hours() / 24)
			if days > 0 {
				totalDays += days
				count++
			}
		}
	}
	if count == 0 {
		return 0
	}
	return totalDays / count
}

func calculateLeadConversionRate(deals []model.Deal, totalLeads int) float64 {
	convertedFromLead := 0
	for _, deal := range deals {
		if deal.LeadID != "" && deal.WonAt != nil {
			convertedFromLead++
		}
	}
	if totalLeads == 0 {
		return 0
	}
	return float64(convertedFromLead) / float64(totalLeads) * 100
}
