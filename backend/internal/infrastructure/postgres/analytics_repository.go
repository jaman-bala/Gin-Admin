package postgres

import (
	"context"
	"gin_auth_service/internal/domain/analytics"

	"github.com/jmoiron/sqlx"
)

// analyticsRepository implements analytics.UserStatsRepository
type analyticsRepository struct {
	db *sqlx.DB
}

// NewAnalyticsRepository creates a new instance of analytics repository.
func NewAnalyticsRepository(db *sqlx.DB) analytics.UserStatsRepository {
	return &analyticsRepository{db: db}
}

// GetUserStats returns aggregated user statistics for dashboard.
func (r *analyticsRepository) GetUserStats(ctx context.Context) (*analytics.UserStats, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE is_active = true) as active,
			COUNT(*) FILTER (WHERE is_active = false) as inactive,
			COUNT(*) FILTER (WHERE role IN ('admin', 'superuser')) as admins,
			COUNT(*) FILTER (WHERE created_at >= DATE_TRUNC('month', NOW())) as new_this_month
		FROM users 
		WHERE deleted_at IS NULL
	`

	var stats analytics.UserStats
	err := r.db.QueryRowContext(ctx, query).Scan(
		&stats.Total,
		&stats.Active,
		&stats.Inactive,
		&stats.Admins,
		&stats.NewThisMonth,
	)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}
