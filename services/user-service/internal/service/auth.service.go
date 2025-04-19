package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/fairuzald/library-system/pkg/constants"
	"github.com/fairuzald/library-system/pkg/logger"
	"github.com/fairuzald/library-system/pkg/middleware"
	"github.com/fairuzald/library-system/pkg/utils"
	"github.com/fairuzald/library-system/services/user-service/internal/entity/dao"
	"github.com/fairuzald/library-system/services/user-service/internal/entity/dto"
	"github.com/fairuzald/library-system/services/user-service/internal/entity/model"
	"github.com/fairuzald/library-system/services/user-service/internal/repository"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type authService struct {
	userRepo   repository.UserRepository
	authRepo   repository.AuthRepository
	jwtAuth    *middleware.JWTAuth
	log        *logger.Logger
	accessExp  time.Duration
	refreshExp time.Duration
}

func NewAuthService(
	userRepo repository.UserRepository,
	authRepo repository.AuthRepository,
	jwtAuth *middleware.JWTAuth,
	log *logger.Logger,
	accessExp time.Duration,
	refreshExp time.Duration,
) AuthService {
	return &authService{
		userRepo:   userRepo,
		authRepo:   authRepo,
		jwtAuth:    jwtAuth,
		log:        log,
		accessExp:  accessExp,
		refreshExp: refreshExp,
	}
}

func (s *authService) Login(ctx context.Context, req *dto.UserLogin) (*dao.TokenResponse, error) {
	user, err := s.userRepo.GetByUsernameOrEmail(ctx, req.UsernameOrEmail)
	if err != nil {
		s.log.Info("Login failed: user not found", zap.String("username_or_email", req.UsernameOrEmail))
		return nil, errors.New(constants.ErrInvalidCredentials)
	}

	if !utils.CheckPasswordHash(req.Password, user.Password) {
		s.log.Info("Login failed: invalid password", zap.String("user_id", user.ID.String()))
		return nil, errors.New(constants.ErrInvalidCredentials)
	}

	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		s.log.Warn("Failed to update last login time", zap.Error(err), zap.String("user_id", user.ID.String()))
	}

	accessToken, err := s.jwtAuth.GenerateToken(
		user.ID.String(),
		user.Email,
		user.Role,
		user.Username,
	)
	if err != nil {
		s.log.Error("Failed to generate access token", zap.Error(err))
		return nil, err
	}

	refreshTokenValue := utils.GenerateRandomString(32)
	refreshTokenExpiry := time.Now().Add(s.refreshExp)

	if err := s.authRepo.StoreRefreshToken(ctx, user.ID, refreshTokenValue, refreshTokenExpiry); err != nil {
		s.log.Error("Failed to store refresh token", zap.Error(err))
		return nil, err
	}

	response := &dao.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenValue,
		TokenType:    constants.TokenTypBearer,
		ExpiresIn:    int64(s.accessExp.Seconds()),
		User:         dao.NewUserResponse(user),
	}

	return response, nil
}

func (s *authService) Register(ctx context.Context, req *dto.UserRegister) (*dao.UserResponse, error) {
	_, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil {
		return nil, errors.New(constants.ErrEmailTaken)
	}

	_, err = s.userRepo.GetByUsername(ctx, req.Username)
	if err == nil {
		return nil, errors.New(constants.ErrUsernameTaken)
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		s.log.Error("Failed to hash password", zap.Error(err))
		return nil, err
	}

	user := model.NewUser(
		req.Email,
		req.Username,
		hashedPassword,
		req.FirstName,
		req.LastName,
		constants.RoleMember,
	)

	if req.Phone != "" {
		user.Phone = req.Phone
	}

	if req.Address != "" {
		user.Address = req.Address
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		s.log.Error("Failed to create user", zap.Error(err))
		return nil, err
	}

	return dao.NewUserResponse(user), nil
}

func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*dao.TokenResponse, error) {
	user, err := s.authRepo.GetUserByRefreshToken(ctx, refreshToken)
	if err != nil {
		s.log.Info("Refresh token failed: invalid token", zap.Error(err))
		return nil, errors.New(constants.ErrInvalidToken)
	}

	accessToken, err := s.jwtAuth.GenerateToken(
		user.ID.String(),
		user.Email,
		user.Role,
		user.Username,
	)
	if err != nil {
		s.log.Error("Failed to generate access token", zap.Error(err))
		return nil, err
	}

	newRefreshTokenValue := utils.GenerateRandomString(32)
	newRefreshTokenExpiry := time.Now().Add(s.refreshExp)

	if err := s.authRepo.StoreRefreshToken(ctx, user.ID, newRefreshTokenValue, newRefreshTokenExpiry); err != nil {
		s.log.Error("Failed to store refresh token", zap.Error(err))
		return &dao.TokenResponse{
			AccessToken: accessToken,
			TokenType:   constants.TokenTypBearer,
			ExpiresIn:   int64(s.accessExp.Seconds()),
		}, nil
	}

	return &dao.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshTokenValue,
		TokenType:    constants.TokenTypBearer,
		ExpiresIn:    int64(s.accessExp.Seconds()),
	}, nil
}

func (s *authService) Logout(ctx context.Context, accessToken string) error {
	claims, err := s.jwtAuth.ValidateToken(accessToken)
	if err != nil {
		return errors.New(constants.ErrInvalidToken)
	}

	expirationTime := time.Unix(claims.ExpiresAt.Unix(), 0)
	timeUntilExpiration := time.Until(expirationTime)

	if timeUntilExpiration > 0 {
		if err := s.authRepo.StoreTokenInBlacklist(ctx, accessToken, timeUntilExpiration); err != nil {
			s.log.Warn("Failed to blacklist token", zap.Error(err))
		}
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return err
	}

	if err := s.authRepo.RevokeRefreshToken(ctx, userID); err != nil {
		s.log.Warn("Failed to revoke refresh token", zap.Error(err))
	}

	return nil
}

func (s *authService) ValidateToken(token string) (*middleware.Claims, error) {
	blacklisted, err := s.authRepo.IsTokenBlacklisted(context.Background(), token)
	if err != nil {
		s.log.Warn("Failed to check token blacklist", zap.Error(err))
	}

	if blacklisted {
		return nil, errors.New(constants.ErrInvalidToken)
	}

	claims, err := s.jwtAuth.ValidateToken(token)
	if err != nil {
		if strings.Contains(err.Error(), "token has expired") {
			return nil, errors.New(constants.ErrExpiredToken)
		}
		return nil, errors.New(constants.ErrInvalidToken)
	}

	return claims, nil
}
