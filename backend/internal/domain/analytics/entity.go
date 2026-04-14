// Package analytics provides domain models for reporting and statistics.
package analytics

// UserStats represents aggregated user statistics for dashboard reporting.
type UserStats struct {
	Total        int64
	Active       int64
	Inactive     int64
	Admins       int64
	NewThisMonth int64
}
