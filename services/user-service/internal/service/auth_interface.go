package service

import (
	"context"

	"github.com/fairuzald/library-system/pkg/middleware"
	"github.com/fairuzald/library-system/services/user-service/internal/entity/dao"
	"github.com/fairuzald/library-system/services/user-service/internal/entity/dto"
)

type AuthService interface {
	Login(ctx context.Context, req *dto.UserLogin) (*dao.TokenResponse, error)
	Register(ctx context.Context, req *dto.UserRegister) (*dao.UserResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*dao.TokenResponse, error)
	Logout(ctx context.Context, accessToken string) error
	ValidateToken(token string) (*middleware.Claims, error)
}
