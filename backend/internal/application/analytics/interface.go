// Package analytics provides application layer for reporting and statistics.
package analytics

import (
	"context"
	"gin_auth_service/internal/domain/analytics"
)

// UseCase provides methods for retrieving aggregated statistics.
type UseCase interface {
	GetUserStats(ctx context.Context) (*analytics.UserStats, error)
}
