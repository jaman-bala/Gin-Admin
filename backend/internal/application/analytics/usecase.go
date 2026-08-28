// Package analytics provides application layer for reporting and statistics.
package analytics

import (
	"context"
	"encoding/json"
	"gin_auth_service/internal/domain/analytics"
	"log/slog"
	"time"
)

const (
	userStatsCacheKey = "cache:analytics:user_stats"
	// GetUserStats runs a full-table COUNT(*) FILTER aggregate; a dashboard
	// showing numbers up to this old is an acceptable trade for not scanning
	// the whole users table on every page load/refresh.
	userStatsCacheTTL = 30 * time.Second
	// cacheCallTimeout bounds each Redis round-trip independently of the
	// caller's context. A dead network path (not a fast "connection
	// refused", but a stalled/partitioned one) must not add several seconds
	// of latency to every dashboard load before falling back to the DB.
	cacheCallTimeout = 150 * time.Millisecond
)

// Cache is the minimal caching contract this use case needs, defined on the
// consumer side (matches the UserReader pattern in application/auth) so the
// application layer depends on a shape, not a concrete Redis client.
// *redis.Cache already satisfies this interface as-is.
type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value any, expiration time.Duration) error
}

type usecase struct {
	repo  analytics.UserStatsRepository
	cache Cache // nil disables caching entirely (e.g. in tests)
}

// NewUseCase creates a new instance of UseCase. cache may be nil, in which
// case every call goes straight to the repository.
func NewUseCase(repo analytics.UserStatsRepository, cache Cache) UseCase {
	return &usecase{repo: repo, cache: cache}
}

// GetUserStats returns aggregated user statistics for the dashboard.
//
// Cache-aside with a short TTL: any cache miss, corrupt payload, or Redis
// outage falls straight through to the database — caching this endpoint must
// never be the reason it goes down. A slower dashboard beats a broken one.
func (uc *usecase) GetUserStats(ctx context.Context) (*analytics.UserStats, error) {
	if uc.cache != nil {
		if stats, ok := uc.getCached(ctx); ok {
			return stats, nil
		}
	}

	stats, err := uc.repo.GetUserStats(ctx)
	if err != nil {
		return nil, err
	}

	if uc.cache != nil {
		uc.setCached(ctx, stats)
	}

	return stats, nil
}

func (uc *usecase) getCached(ctx context.Context) (*analytics.UserStats, bool) {
	ctx, cancel := context.WithTimeout(ctx, cacheCallTimeout)
	defer cancel()

	raw, err := uc.cache.Get(ctx, userStatsCacheKey)
	if err != nil {
		// Cache miss and Redis-unavailable both land here — neither is
		// fatal, both mean "go read the database instead".
		return nil, false
	}
	var stats analytics.UserStats
	if err := json.Unmarshal([]byte(raw), &stats); err != nil {
		slog.Warn("analytics: discarding corrupt cached user stats", "error", err)
		return nil, false
	}
	return &stats, true
}

func (uc *usecase) setCached(ctx context.Context, stats *analytics.UserStats) {
	raw, err := json.Marshal(stats)
	if err != nil {
		slog.Warn("analytics: failed to marshal user stats for cache", "error", err)
		return
	}

	ctx, cancel := context.WithTimeout(ctx, cacheCallTimeout)
	defer cancel()

	// Best-effort: a failed cache write must not fail the request that
	// already has a good result from the database.
	if err := uc.cache.Set(ctx, userStatsCacheKey, raw, userStatsCacheTTL); err != nil {
		slog.Warn("analytics: failed to write user stats cache", "error", err)
	}
}
