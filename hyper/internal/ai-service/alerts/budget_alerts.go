package alerts

import (
	"context"
	"fmt"
	"time"

	"hyper/internal/storage"
)

// BudgetAlertManager manages budget monitoring and alerts
type BudgetAlertManager struct {
	storage storage.MetricsStorage
}

// NewBudgetAlertManager creates a new budget alert manager
func NewBudgetAlertManager(storage storage.MetricsStorage) *BudgetAlertManager {
	return &BudgetAlertManager{
		storage: storage,
	}
}

// BudgetConfig represents budget configuration for a user
type BudgetConfig struct {
	DailyBudget      float64 `json:"dailyBudget"`
	MonthlyBudget    float64 `json:"monthlyBudget"`
	PerRequestBudget float64 `json:"perRequestBudget"`
	AlertThreshold   float64 `json:"alertThreshold"` // percentage (e.g., 80 for 80%)
}

// CheckBudgets checks if any budgets have been exceeded
func (bam *BudgetAlertManager) CheckBudgets(ctx context.Context, userID, companyID string, config BudgetConfig) ([]storage.BudgetAlert, error) {
	alerts := []storage.BudgetAlert{}

	// Check daily budget
	if config.DailyBudget > 0 {
		today := time.Now()
		startOfDay := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
		endOfDay := startOfDay.Add(24 * time.Hour)

		dailyAlert, err := bam.checkDailyBudget(ctx, userID, companyID, startOfDay, endOfDay, config)
		if err != nil {
			return nil, fmt.Errorf("failed to check daily budget: %w", err)
		}
		if dailyAlert != nil {
			alerts = append(alerts, *dailyAlert)
		}
	}

	// Check monthly budget
	if config.MonthlyBudget > 0 {
		today := time.Now()
		startOfMonth := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location())
		endOfMonth := startOfMonth.AddDate(0, 1, 0)

		monthlyAlert, err := bam.checkMonthlyBudget(ctx, userID, companyID, startOfMonth, endOfMonth, config)
		if err != nil {
			return nil, fmt.Errorf("failed to check monthly budget: %w", err)
		}
		if monthlyAlert != nil {
			alerts = append(alerts, *monthlyAlert)
		}
	}

	return alerts, nil
}

// checkDailyBudget checks if daily budget has been exceeded
func (bam *BudgetAlertManager) checkDailyBudget(ctx context.Context, userID, companyID string, startTime, endTime time.Time, config BudgetConfig) (*storage.BudgetAlert, error) {
	costMetrics, err := bam.storage.GetCostMetrics(ctx, userID, startTime, endTime)
	if err != nil {
		return nil, err
	}

	var totalCost float64
	for _, cm := range costMetrics {
		totalCost += cm.TotalCost
	}

	percentageUsed := (totalCost / config.DailyBudget) * 100

	// Create alert if threshold exceeded
	if percentageUsed >= config.AlertThreshold {
		severity := "warning"
		if percentageUsed >= 100 {
			severity = "critical"
		}

		alert := &storage.BudgetAlert{
			UserID:        userID,
			CompanyID:     companyID,
			AlertType:     "daily",
			Threshold:     config.DailyBudget,
			CurrentValue:  totalCost,
			PercentageUsed: percentageUsed,
			Severity:      severity,
			Message:       fmt.Sprintf("Daily budget alert: %.2f%% of daily budget used ($%.2f / $%.2f)", percentageUsed, totalCost, config.DailyBudget),
			IsResolved:    false,
			CreatedAt:     time.Now(),
		}

		// Save alert
		if err := bam.storage.SaveBudgetAlert(ctx, alert); err != nil {
			return nil, fmt.Errorf("failed to save budget alert: %w", err)
		}

		return alert, nil
	}

	return nil, nil
}

// checkMonthlyBudget checks if monthly budget has been exceeded
func (bam *BudgetAlertManager) checkMonthlyBudget(ctx context.Context, userID, companyID string, startTime, endTime time.Time, config BudgetConfig) (*storage.BudgetAlert, error) {
	costMetrics, err := bam.storage.GetCostMetrics(ctx, userID, startTime, endTime)
	if err != nil {
		return nil, err
	}

	var totalCost float64
	for _, cm := range costMetrics {
		totalCost += cm.TotalCost
	}

	percentageUsed := (totalCost / config.MonthlyBudget) * 100

	// Create alert if threshold exceeded
	if percentageUsed >= config.AlertThreshold {
		severity := "warning"
		if percentageUsed >= 100 {
			severity = "critical"
		}

		alert := &storage.BudgetAlert{
			UserID:        userID,
			CompanyID:     companyID,
			AlertType:     "monthly",
			Threshold:     config.MonthlyBudget,
			CurrentValue:  totalCost,
			PercentageUsed: percentageUsed,
			Severity:      severity,
			Message:       fmt.Sprintf("Monthly budget alert: %.2f%% of monthly budget used ($%.2f / $%.2f)", percentageUsed, totalCost, config.MonthlyBudget),
			IsResolved:    false,
			CreatedAt:     time.Now(),
		}

		// Save alert
		if err := bam.storage.SaveBudgetAlert(ctx, alert); err != nil {
			return nil, fmt.Errorf("failed to save budget alert: %w", err)
		}

		return alert, nil
	}

	return nil, nil
}

// GetActiveAlerts retrieves all active (unresolved) alerts for a user
func (bam *BudgetAlertManager) GetActiveAlerts(ctx context.Context, userID string) ([]*storage.BudgetAlert, error) {
	alerts, err := bam.storage.GetBudgetAlerts(ctx, userID, 100)
	if err != nil {
		return nil, err
	}

	// Filter for unresolved alerts
	activeAlerts := []*storage.BudgetAlert{}
	for _, alert := range alerts {
		if !alert.IsResolved {
			activeAlerts = append(activeAlerts, alert)
		}
	}

	return activeAlerts, nil
}

// ResolveAlert marks an alert as resolved
func (bam *BudgetAlertManager) ResolveAlert(ctx context.Context, alertID string) error {
	// This would require updating the alert in storage
	// For now, we'll just return a placeholder
	return nil
}

// AnomalyDetection detects unusual spending patterns
type AnomalyDetection struct {
	IsAnomaly      bool    `json:"isAnomaly"`
	Severity       string  `json:"severity"` // "low", "medium", "high"
	Description    string  `json:"description"`
	ExpectedCost   float64 `json:"expectedCost"`
	ActualCost     float64 `json:"actualCost"`
	Deviation      float64 `json:"deviation"` // percentage
}

// DetectAnomalies detects unusual spending patterns
func (bam *BudgetAlertManager) DetectAnomalies(ctx context.Context, userID string, currentCost float64) (*AnomalyDetection, error) {
	// Get historical data (last 7 days)
	today := time.Now()
	sevenDaysAgo := today.AddDate(0, 0, -7)

	costMetrics, err := bam.storage.GetCostMetrics(ctx, userID, sevenDaysAgo, today)
	if err != nil {
		return nil, err
	}

	if len(costMetrics) == 0 {
		// Not enough historical data
		return &AnomalyDetection{
			IsAnomaly: false,
		}, nil
	}

	// Calculate average daily cost
	var totalCost float64
	for _, cm := range costMetrics {
		totalCost += cm.TotalCost
	}
	averageDailyCost := totalCost / 7

	// Calculate deviation
	deviation := ((currentCost - averageDailyCost) / averageDailyCost) * 100

	detection := &AnomalyDetection{
		ExpectedCost: averageDailyCost,
		ActualCost:   currentCost,
		Deviation:    deviation,
	}

	// Determine if it's an anomaly (>50% deviation)
	if deviation > 50 {
		detection.IsAnomaly = true
		if deviation > 100 {
			detection.Severity = "high"
			detection.Description = fmt.Sprintf("Spending is %.0f%% higher than average. This may indicate unusual activity.", deviation)
		} else {
			detection.Severity = "medium"
			detection.Description = fmt.Sprintf("Spending is %.0f%% higher than average.", deviation)
		}
	} else if deviation < -50 {
		detection.IsAnomaly = true
		detection.Severity = "low"
		detection.Description = fmt.Sprintf("Spending is %.0f%% lower than average.", -deviation)
	}

	return detection, nil
}
