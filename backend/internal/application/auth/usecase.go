package auth

import (
	"context"
	stdErrors "errors"
	"fmt"
	"gin_auth_service/internal/application/token"
	"gin_auth_service/internal/application/user"
	domainUser "gin_auth_service/internal/domain/user"
	"gin_auth_service/internal/pkg/hash"
	"gin_auth_service/internal/pkg/utils"
	"gin_auth_service/pkg/errors"
	"time"

	"uuid"
)

// dummyHash is compared against when the user is not found, equalizing
// response time with the real bcrypt check to prevent user enumeration
// via timing analysis.
var dummyHash, _ = hash.HashPassword("dummy-timing-equalizer")

// UserReader is the minimal interface auth needs from the user domain.
// Defined here (consumer side) to avoid importing the full user.UseCase.
type UserReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*user.UserResponseDTO, error)
}

type usecase struct {
	userRepo      domainUser.Repository
	tokenService  token.UseCase
	userReader    UserReader
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func NewUseCase(
	userRepo domainUser.Repository,
	tokenService token.UseCase,
	userReader UserReader,
	accessExpiry time.Duration,
	refreshExpiry time.Duration,
) UseCase {
	return &usecase{
		userRepo:      userRepo,
		tokenService:  tokenService,
		userReader:    userReader,
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
	}
}

func (uc *usecase) Login(ctx context.Context, req LoginRequestDTO) (*LoginResponseDTO, error) {
	req.Phone = utils.NormalizePhone(req.Phone)
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

	userDTO, err := uc.userReader.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !userDTO.IsActive {
		return nil, errors.ErrAccountBlocked
	}

	// Invalidate the old refresh token before issuing a new pair.
	// If this fails, abort — issuing a new pair while the old one remains valid
	// would defeat the purpose of rotation.
	if err := uc.tokenService.BlacklistToken(ctx, refreshToken, info.ExpiresAt); err != nil {
		return nil, fmt.Errorf("failed to invalidate refresh token: %w", err)
	}

	pair, err := uc.tokenService.GenerateTokenPair(ctx, userDTO.ID.String(), string(userDTO.Role), userDTO.IsActive, uc.accessExpiry, uc.refreshExpiry)
	if err != nil {
		return nil, fmt.Errorf("token generation error: %w", err)
	}

	return &TokenResponseDTO{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		User:         *userDTO,
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