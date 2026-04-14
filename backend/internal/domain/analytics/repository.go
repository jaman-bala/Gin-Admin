package analytics

import "context"

// UserStatsRepository defines the interface for user statistics queries.
type UserStatsRepository interface {
	GetUserStats(ctx context.Context) (*UserStats, error)
}
