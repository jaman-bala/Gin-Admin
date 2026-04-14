// Package analytics provides application layer for reporting and statistics.
package analytics

import (
	"context"
	"gin_auth_service/internal/domain/analytics"
)


type usecase struct {
	repo analytics.UserStatsRepository
}

// NewUseCase creates a new instance of UseCase.
func NewUseCase(repo analytics.UserStatsRepository) UseCase {
	return &usecase{
		repo: repo,
	}
}

// GetUserStats returns aggregated user statistics for dashboard.
func (uc *usecase) GetUserStats(ctx context.Context) (*analytics.UserStats, error) {
	return uc.repo.GetUserStats(ctx)
}
