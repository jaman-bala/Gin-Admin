package auth

import (
	"context"
	stdErrors "errors"
	"fmt"
	"gin_auth_service/internal/application/token"
	domainUser "gin_auth_service/internal/domain/user"
	"gin_auth_service/internal/pkg/hash"
	"gin_auth_service/internal/pkg/utils"
	"gin_auth_service/pkg/errors"
	"log/slog"
	"time"

	"uuid"
)

// dummyHash is compared against when the user is not found, equalizing
// response time with the real bcrypt check to prevent user enumeration
// via timing analysis.
var dummyHash, _ = hash.HashPassword("dummy-timing-equalizer")

// PhotoResolver resolves a stored photo object key to a browsable URL.
// Defined here (consumer side) so auth depends on a one-method shape, not on
// application/file's full UseCase or application/user's DTOs; *file.usecase
// already satisfies this as-is.
type PhotoResolver interface {
	GetFullURL(ctx context.Context, objectName string) (string, error)
}

// AttemptLimiter counts login attempts per account. Defined here (consumer
// side); *redis.Cache satisfies it as-is.
type AttemptLimiter interface {
	// Increment atomically takes a slot and returns the counter after it.
	Increment(ctx context.Context, key string, window time.Duration) (int64, error)
	// Release gives that slot back.
	Release(ctx context.Context, key string) error
}

const (
	// accountMaxFailures caps failed logins per account per window,
	// regardless of how many IPs they come from: the defense against a
	// distributed guess that the per-IP limiter cannot see.
	//
	// Trade-off: anyone who knows a phone number can keep that account's
	// login locked by failing on purpose. Kept high enough (10) that an
	// honest typo streak never trips it, short enough (15 min) to bound the
	// damage.
	accountMaxFailures int64 = 10
	accountWindow            = 15 * time.Minute
)

type usecase struct {
	limiter       AttemptLimiter
	userRepo      domainUser.Repository
	tokenService  token.UseCase
	photoResolver PhotoResolver
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func NewUseCase(
	limiter AttemptLimiter,
	userRepo domainUser.Repository,
	tokenService token.UseCase,
	photoResolver PhotoResolver,
	accessExpiry time.Duration,
	refreshExpiry time.Duration,
) UseCase {
	return &usecase{
		limiter:       limiter,
		userRepo:      userRepo,
		tokenService:  tokenService,
		photoResolver: photoResolver,
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
	}
}

// Login enforces a per-account failure budget around the credential check.
// The slot is reserved before the password is looked at (so concurrent guesses
// cannot overshoot the limit) and given back only on success, which means
// failures accumulate and a success never erases them. The budget applies to
// every phone number, existing or not, so a 429 reveals nothing about which
// accounts exist.
func (uc *usecase) Login(ctx context.Context, req LoginRequestDTO) (*LoginResponseDTO, error) {
	req.Phone = utils.NormalizePhone(req.Phone)

	key := "rate_limit:login:phone:" + req.Phone
	reserved := false
	if uc.limiter != nil {
		n, err := uc.limiter.Increment(ctx, key, accountWindow)
		switch {
		case err != nil:
			// Fail open: a Redis outage must not lock everyone out.
			slog.Warn("account login limiter unavailable, failing open", "error", err)
		case n > accountMaxFailures:
			return nil, errors.ErrTooManyAttempts
		default:
			reserved = true
		}
	}

	resp, err := uc.authenticate(ctx, req)
	if err == nil && reserved {
		if relErr := uc.limiter.Release(ctx, key); relErr != nil {
			slog.Warn("account login limiter release failed", "error", relErr)
		}
	}
	return resp, err
}

func (uc *usecase) authenticate(ctx context.Context, req LoginRequestDTO) (*LoginResponseDTO, error) {
	u, err := uc.userRepo.FindByPhone(ctx, req.Phone)
	if err != nil {
		if stdErrors.Is(err, errors.ErrUserNotFound) {
			_ = hash.CheckPassword(dummyHash, req.Password)
			return nil, errors.ErrInvalidCredentials
		}
		// Infrastructure failure must surface as 500, not "wrong password".
		return nil, fmt.Errorf("login: find user: %w", err)
	}
	if !u.IsActive {
		return nil, errors.ErrAccountBlocked
	}
	if err := u.CheckPassword(req.Password); err != nil {
		return nil, errors.ErrInvalidCredentials
	}

	pair, err := uc.tokenService.GenerateTokenPair(ctx, u.ID.String(), string(u.Role), u.IsActive, uc.accessExpiry, uc.refreshExpiry)
	if err != nil {
		return nil, fmt.Errorf("token generation error: %w", err)
	}

	return &LoginResponseDTO{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		Message:      "Success authorization",
	}, nil
}

func (uc *usecase) Logout(ctx context.Context, accessToken, refreshToken string) error {
	info, err := uc.tokenService.GetTokenInfo(ctx, accessToken)
	if err != nil {
		return err
	}
	if info.Type != "access" {
		return errors.ErrInvalidToken
	}
	if err := uc.tokenService.BlacklistToken(ctx, accessToken, info.ExpiresAt); err != nil {
		return err
	}

	// Best-effort: blacklist the refresh token if provided.
	// Access token is already invalidated above, so we don't fail the logout
	// if the refresh token is missing, expired, or already blacklisted.
	if refreshToken != "" {
		if refreshInfo, err := uc.tokenService.GetTokenInfo(ctx, refreshToken); err == nil && refreshInfo.Type == "refresh" {
			_ = uc.tokenService.BlacklistToken(ctx, refreshToken, refreshInfo.ExpiresAt)
		}
	}

	return nil
}

func (uc *usecase) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponseDTO, error) {
	info, err := uc.tokenService.GetTokenInfo(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	if info.Type != "refresh" {
		return nil, errors.ErrInvalidToken
	}

	userID, err := uuid.Parse(info.UserID)
	if err != nil {
		return nil, errors.ErrInvalidUUID
	}

	u, err := uc.userRepo.GetID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !u.IsActive {
		return nil, errors.ErrAccountBlocked
	}

	// Invalidate the old refresh token before issuing a new pair.
	// If this fails, abort — issuing a new pair while the old one remains valid
	// would defeat the purpose of rotation.
	if err := uc.tokenService.BlacklistToken(ctx, refreshToken, info.ExpiresAt); err != nil {
		return nil, fmt.Errorf("failed to invalidate refresh token: %w", err)
	}

	pair, err := uc.tokenService.GenerateTokenPair(ctx, u.ID.String(), string(u.Role), u.IsActive, uc.accessExpiry, uc.refreshExpiry)
	if err != nil {
		return nil, fmt.Errorf("token generation error: %w", err)
	}

	var photoURL string
	if u.Photo != "" {
		if resolved, err := uc.photoResolver.GetFullURL(ctx, u.Photo); err == nil {
			photoURL = resolved
		}
		// A presign failure must not fail the refresh — the client just gets
		// an empty photo URL and can re-fetch the profile separately.
	}

	return &TokenResponseDTO{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		User:         AuthUserFromDomain(u, photoURL),
		ExpiresAt:    pair.AccessExpiry,
		Message:      "Token refreshed successfully",
	}, nil
}

// GetUserInfoFromToken decodes user identity from a valid access token without hitting the database.
// Freshness of is_active relies on token expiry; immediate revocation must be handled via blacklisting.
func (uc *usecase) GetUserInfoFromToken(ctx context.Context, tokenString string) (*domainUser.User, error) {
	info, err := uc.tokenService.GetTokenInfo(ctx, tokenString)
	if err != nil {
		return nil, err
	}
	if info.Type != "access" {
		return nil, errors.ErrInvalidToken
	}
	userID, err := uuid.Parse(info.UserID)
	if err != nil {
		return nil, errors.ErrInvalidUUID
	}
	return &domainUser.User{
		ID:       userID,
		Role:     domainUser.Role(info.Role),
		IsActive: info.IsActive,
	}, nil
}
