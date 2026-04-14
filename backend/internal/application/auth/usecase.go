package auth

import (
	"context"
	"fmt"
	"gin_auth_service/config"
	"gin_auth_service/internal/application/file"
	"gin_auth_service/internal/application/token"
	"gin_auth_service/internal/application/user"
	domainUser "gin_auth_service/internal/domain/user"
	"gin_auth_service/pkg/errors"
	"log"
	"time"

	jwtv4 "github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

type usecase struct {
	userRepo     domainUser.Repository
	tokenService token.UseCase
	fileService  file.UseCase
	cfg          *config.Config
}

// NewUseCase creates a new instance of UseCase.
func NewUseCase(userRepo domainUser.Repository, tokenService token.UseCase, fileService file.UseCase, cfg *config.Config) UseCase {
	return &usecase{
		userRepo:     userRepo,
		tokenService: tokenService,
		fileService:  fileService,
		cfg:          cfg,
	}
}

func (uc *usecase) Login(ctx context.Context, req LoginRequestDTO) (*LoginResponseDTO, error) {
	u, err := uc.userRepo.FindByPhone(ctx, req.Phone)
	if err != nil {
		return nil, errors.ErrInvalidCredentials
	}

	if !u.IsActive {
		return nil, errors.ErrAccountBlocked
	}

	if err := u.CheckPassword(req.Password); err != nil {
		return nil, errors.ErrInvalidCredentials
	}

	accessToken, refreshToken, _, err := uc.generateJWTToken(u)
	if err != nil {
		return nil, fmt.Errorf("token generation error: %w", err)
	}

	return &LoginResponseDTO{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Message:      "Success authorization",
	}, nil
}

func (uc *usecase) Logout(ctx context.Context, tokenString string) error {
	info, err := uc.tokenService.GetTokenInfo(ctx, tokenString)
	if err != nil {
		return err
	}

	return uc.tokenService.BlacklistToken(ctx, tokenString, info.ExpiresAt)
}

func (uc *usecase) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponseDTO, error) {
	info, err := uc.tokenService.GetTokenInfo(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	if info.Type != "refresh" {
		return nil, errors.ErrInvalidToken
	}

	userID, _ := uuid.Parse(info.UserID)
	u, err := uc.userRepo.GetID(ctx, userID)
	if err != nil {
		return nil, err
	}

	accessToken, _, expiresAt, err := uc.generateJWTToken(u)
	if err != nil {
		return nil, err
	}

	var userResp user.UserResponseDTO
	userResp.FromModel(u)
	uc.enrichDTO(ctx, &userResp)

	return &TokenResponseDTO{
		AccessToken: accessToken,
		User:        userResp,
		ExpiresAt:   expiresAt,
		Message:     "Token refreshed successfully",
	}, nil
}

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

	u, err := uc.userRepo.GetID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func (uc *usecase) generateJWTToken(u *domainUser.User) (string, string, time.Time, error) {
	accessExpiry := time.Now().Add(uc.cfg.JWT.Expiry)
	refreshExpiry := time.Now().Add(uc.cfg.JWT.RefreshExpiry)

	gen := func(exp time.Time, t string) (string, error) {
		claims := jwtv4.MapClaims{
			"user_id": u.ID.String(),
			"role":    string(u.Role),
			"exp":     exp.Unix(),
			"type":    t,
			"jti":     uuid.New().String(),
			"iat":     time.Now().Unix(),
		}
		token := jwtv4.NewWithClaims(jwtv4.SigningMethodHS256, claims)
		return token.SignedString([]byte(uc.cfg.JWT.Secret))
	}

	at, err := gen(accessExpiry, "access")
	if err != nil {
		return "", "", time.Time{}, err
	}
	rt, err := gen(refreshExpiry, "refresh")
	if err != nil {
		return "", "", time.Time{}, err
	}

	return at, rt, accessExpiry, nil
}

// GetMe retrieves current user profile by ID and returns DTO without sensitive data
func (uc *usecase) GetMe(ctx context.Context, userID uuid.UUID) (*user.UserResponseDTO, error) {
	u, err := uc.userRepo.GetID(ctx, userID)
	if err != nil {
		return nil, errors.ErrUserNotFound
	}

	var dto user.UserResponseDTO
	dto.FromModel(u)
	uc.enrichDTO(ctx, &dto)
	return &dto, nil
}

func (uc *usecase) enrichDTO(ctx context.Context, dto *user.UserResponseDTO) {
	if dto.Photo != "" {
		url, err := uc.fileService.GetFullURL(ctx, dto.Photo)
		if err == nil {
			dto.Photo = url
			log.Printf("[DEBUG] GetFullURL success: %s", url)
		} else {
			log.Printf("[ERROR] GetFullURL failed for photo '%s': %v", dto.Photo, err)
		}
	}
}
