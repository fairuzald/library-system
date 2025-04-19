package service

import (
	"context"

	"github.com/fairuzald/library-system/services/user-service/internal/entity/dao"
	"github.com/fairuzald/library-system/services/user-service/internal/entity/dto"
	"github.com/google/uuid"
)

type UserService interface {
	CreateUser(ctx context.Context, req *dto.UserCreate) (*dao.UserResponse, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*dao.UserResponse, error)
	GetUserByEmail(ctx context.Context, email string) (*dao.UserResponse, error)
	GetUserByUsername(ctx context.Context, username string) (*dao.UserResponse, error)
	UpdateUser(ctx context.Context, id uuid.UUID, req *dto.UserUpdate) (*dao.UserResponse, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
	ListUsers(ctx context.Context, filter *dto.UserFilter) (*dao.UserListResponse, error)
	ChangePassword(ctx context.Context, id uuid.UUID, req *dto.ChangePassword) error
}
