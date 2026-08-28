package user

import (
	"context"
	"encoding/json"
	"fmt"
	"gin_auth_service/internal/application/file"
	domainUser "gin_auth_service/internal/domain/user"
	"gin_auth_service/internal/pkg/hash"
	"gin_auth_service/internal/pkg/utils"
	"gin_auth_service/pkg/errors"
	"log/slog"
	"time"

	"uuid"
)

const (
	userListCacheKeyPrefix = "cache:users:list"
	userListVersionKey     = "cache:users:list:version"
	// userListCacheTTL is a backstop, not the primary invalidation path:
	// every mutation bumps userListVersionKey, which changes the cache key
	// and makes prior entries unreachable immediately. The TTL only bounds
	// staleness in case a version bump is ever missed (e.g. Redis outage
	// during a write) and just cleans up unreachable entries afterwards.
	userListCacheTTL = 15 * time.Second
	// versionWindow is how long the version counter itself survives if
	// untouched. Losing it just resets the epoch — cached pages tagged
	// with the old version stop matching and expire on their own via
	// userListCacheTTL, so there's no correctness issue, only a cache miss.
	versionWindow = 24 * time.Hour
	// cacheCallTimeout bounds each Redis round-trip independently of the
	// caller's context, matching the analytics usecase: a stalled Redis
	// connection must not add latency to a request that would otherwise be
	// served fine from the database.
	cacheCallTimeout = 150 * time.Millisecond
)

// Cache is the minimal caching contract this use case needs, defined on the
// consumer side (matches the pattern in application/analytics) so the
// application layer depends on a shape, not a concrete Redis client.
// *redis.Cache already satisfies this interface as-is.
type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value any, expiration time.Duration) error
	// Increment atomically increments key and (re)arms its TTL only when the
	// key is first created — reused here purely as a version counter, not
	// for rate limiting.
	Increment(ctx context.Context, key string, window time.Duration) (int64, error)
}

type usecase struct {
	repo        domainUser.Repository
	fileService file.UseCase
	cache       Cache // nil disables caching entirely (e.g. in tests)
}

// NewUseCase creates a new instance of UseCase. cache may be nil, in which
// case every call goes straight to the repository.
func NewUseCase(repo domainUser.Repository, fileService file.UseCase, cache Cache) UseCase {
	return &usecase{repo: repo, fileService: fileService, cache: cache}
}

// GetAll returns a page of users.
//
// Only the unfiltered listing (no search, no is_active filter) is cached —
// that's the default view every admin lands on when opening the Users page
// (see migration 00006's ix_users_live_created_at), and the one query worth
// sparing the DB. A search or filtered query always goes straight to the
// repository: results are per-input, rarely repeated, and already served by
// the trigram/partial indexes from migration 00005.
func (uc *usecase) GetAll(ctx context.Context, page, limit int, search string, isActive *bool) (*UserListResponse, error) {
	cacheable := uc.cache != nil && search == "" && isActive == nil

	if cacheable {
		if resp, ok := uc.getCachedList(ctx, page, limit); ok {
			return resp, nil
		}
	}

	params := domainUser.FilterParams{
		Page:     page,
		Limit:    limit,
		Search:   search,
		IsActive: isActive,
	}

	users, total, err := uc.repo.GetWithFilters(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	dtos := make([]*UserResponseDTO, 0, len(users))
	for _, u := range users {
		var dto UserResponseDTO
		dto.FromModel(u)
		uc.enrichDTO(ctx, &dto)
		dtos = append(dtos, &dto)
	}

	resp := &UserListResponse{
		Users: dtos,
		Total: total,
		Page:  page,
		Limit: limit,
	}

	if cacheable {
		uc.setCachedList(ctx, page, limit, resp)
	}

	return resp, nil
}

// listVersion reads the current cache epoch for the user list. A miss or
// Redis error is treated as version 0 — safe, since it just means the next
// invalidation (Increment) moves everyone from 0 to 1, still invalidating
// whatever was cached under the zero value.
func (uc *usecase) listVersion(ctx context.Context) int64 {
	ctx, cancel := context.WithTimeout(ctx, cacheCallTimeout)
	defer cancel()

	raw, err := uc.cache.Get(ctx, userListVersionKey)
	if err != nil {
		return 0
	}
	var v int64
	if _, err := fmt.Sscanf(raw, "%d", &v); err != nil {
		return 0
	}
	return v
}

func (uc *usecase) listCacheKey(version int64, page, limit int) string {
	return fmt.Sprintf("%s:v%d:page:%d:limit:%d", userListCacheKeyPrefix, version, page, limit)
}

func (uc *usecase) getCachedList(ctx context.Context, page, limit int) (*UserListResponse, bool) {
	key := uc.listCacheKey(uc.listVersion(ctx), page, limit)

	callCtx, cancel := context.WithTimeout(ctx, cacheCallTimeout)
	defer cancel()

	raw, err := uc.cache.Get(callCtx, key)
	if err != nil {
		// Cache miss and Redis-unavailable both land here — neither is
		// fatal, both mean "go read the database instead".
		return nil, false
	}
	var resp UserListResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		slog.Warn("user: discarding corrupt cached user list", "error", err)
		return nil, false
	}
	return &resp, true
}

func (uc *usecase) setCachedList(ctx context.Context, page, limit int, resp *UserListResponse) {
	raw, err := json.Marshal(resp)
	if err != nil {
		slog.Warn("user: failed to marshal user list for cache", "error", err)
		return
	}

	key := uc.listCacheKey(uc.listVersion(ctx), page, limit)

	callCtx, cancel := context.WithTimeout(ctx, cacheCallTimeout)
	defer cancel()

	// Best-effort: a failed cache write must not fail a request that
	// already has a good result from the database.
	if err := uc.cache.Set(callCtx, key, raw, userListCacheTTL); err != nil {
		slog.Warn("user: failed to write user list cache", "error", err)
	}
}

// invalidateListCache bumps the list cache epoch so every previously cached
// page becomes unreachable immediately. Called after any write that changes
// what the unfiltered listing would return (create, patch, delete).
// Best-effort: a failed invalidation just means the cache falls back to its
// userListCacheTTL backstop instead of invalidating instantly.
func (uc *usecase) invalidateListCache(ctx context.Context) {
	if uc.cache == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, cacheCallTimeout)
	defer cancel()

	if _, err := uc.cache.Increment(ctx, userListVersionKey, versionWindow); err != nil {
		slog.Warn("user: failed to invalidate user list cache", "error", err)
	}
}

func (uc *usecase) Create(ctx context.Context, req UserRequestDTO, photo *file.FileUpload) (*UserResponseDTO, error) {
	req.Phone = utils.NormalizePhone(req.Phone)
	hashedPassword, err := hash.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("error hashing password: %w", err)
	}

	u := &domainUser.User{
		ID:         uuid.NewV7(),
		FirstName:  req.FirstName,
		LastName:   req.LastName,
		MiddleName: req.MiddleName,
		Phone:      req.Phone,
		Password:   hashedPassword,
		Role:       domainUser.Role(req.Role),
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if photo != nil {
		filePath, err := uc.fileService.UploadFile(ctx, photo, "user")
		if err != nil {
			return nil, fmt.Errorf("error saving photo: %w", err)
		}
		u.Photo = filePath
	}

	// Unique phone constraint is enforced by the DB; the repo maps the
	// violation to ErrConflict, which propagates here without wrapping.
	if err := uc.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	uc.invalidateListCache(ctx)

	var dto UserResponseDTO
	dto.FromModel(u)
	uc.enrichDTO(ctx, &dto)
	return &dto, nil
}

func (uc *usecase) GetByID(ctx context.Context, id uuid.UUID) (*UserResponseDTO, error) {
	u, err := uc.repo.GetID(ctx, id)
	if err != nil {
		return nil, err
	}
	var dto UserResponseDTO
	dto.FromModel(u)
	uc.enrichDTO(ctx, &dto)
	return &dto, nil
}

func (uc *usecase) GetByPhone(ctx context.Context, phone string) (*UserResponseDTO, error) {
	phone = utils.NormalizePhone(phone)
	u, err := uc.repo.FindByPhone(ctx, phone)
	if err != nil {
		return nil, err
	}
	var dto UserResponseDTO
	dto.FromModel(u)
	uc.enrichDTO(ctx, &dto)
	return &dto, nil
}

func (uc *usecase) GetMe(ctx context.Context, id uuid.UUID) (*UserResponseDTO, error) {
	u, err := uc.repo.GetID(ctx, id)
	if err != nil {
		return nil, err
	}
	var dto UserResponseDTO
	dto.FromModel(u)
	uc.enrichDTO(ctx, &dto)
	return &dto, nil
}

// PatchSelf updates the caller's own profile. Role and IsActive are excluded
// from req — users cannot escalate their own privileges.
func (uc *usecase) PatchSelf(ctx context.Context, id uuid.UUID, req UserSelfUpdateDTO, photo *file.FileUpload) (*UserResponseDTO, error) {
	if id == uuid.Nil() {
		return nil, errors.ErrInvalidUUID
	}

	if req.Phone != nil {
		normalized := utils.NormalizePhone(*req.Phone)
		req.Phone = &normalized
	}

	u, err := uc.repo.GetID(ctx, id)
	if err != nil {
		return nil, err
	}

	req.ApplyToModel(u)

	if req.Password != nil && *req.Password != "" {
		// A valid access token alone must not allow a password change.
		if req.CurrentPassword == nil || u.CheckPassword(*req.CurrentPassword) != nil {
			return nil, errors.ErrInvalidCredentials
		}
		hashed, err := hash.HashPassword(*req.Password)
		if err != nil {
			return nil, fmt.Errorf("error hashing password: %w", err)
		}
		u.Password = hashed
	}

	if photo != nil {
		filePath, err := uc.fileService.UploadFile(ctx, photo, "user")
		if err != nil {
			return nil, fmt.Errorf("error saving photo: %w", err)
		}
		u.Photo = filePath
	}

	u.UpdatedAt = time.Now()
	if err := uc.repo.Patch(ctx, u); err != nil {
		return nil, err
	}
	uc.invalidateListCache(ctx)

	var dto UserResponseDTO
	dto.FromModel(u)
	uc.enrichDTO(ctx, &dto)
	return &dto, nil
}

// Patch is the admin-only update; it allows changing Role and IsActive.
func (uc *usecase) Patch(ctx context.Context, id uuid.UUID, req UserUpdateDTO, photo *file.FileUpload) (*UserResponseDTO, error) {
	if id == uuid.Nil() {
		return nil, errors.ErrInvalidUUID
	}

	if req.Phone != nil {
		normalized := utils.NormalizePhone(*req.Phone)
		req.Phone = &normalized
	}

	u, err := uc.repo.GetID(ctx, id)
	if err != nil {
		return nil, err
	}

	req.ApplyToModel(u)

	if req.Password != nil && *req.Password != "" {
		hashed, err := hash.HashPassword(*req.Password)
		if err != nil {
			return nil, fmt.Errorf("error hashing password: %w", err)
		}
		u.Password = hashed
	}

	if photo != nil {
		filePath, err := uc.fileService.UploadFile(ctx, photo, "user")
		if err != nil {
			return nil, fmt.Errorf("error saving photo: %w", err)
		}
		u.Photo = filePath
	}

	u.UpdatedAt = time.Now()
	if err := uc.repo.Patch(ctx, u); err != nil {
		return nil, err
	}
	uc.invalidateListCache(ctx)

	var dto UserResponseDTO
	dto.FromModel(u)
	uc.enrichDTO(ctx, &dto)
	return &dto, nil
}

func (uc *usecase) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil() {
		return errors.ErrInvalidUUID
	}
	if err := uc.repo.Delete(ctx, id); err != nil {
		return err
	}
	uc.invalidateListCache(ctx)
	return nil
}

func (uc *usecase) enrichDTO(ctx context.Context, dto *UserResponseDTO) {
	if dto.Photo == "" {
		return
	}
	url, err := uc.fileService.GetFullURL(ctx, dto.Photo)
	if err != nil {
		slog.Error("enrichDTO: failed to generate photo URL", "objectName", dto.Photo, "error", err)
		return
	}
	dto.Photo = url
}
