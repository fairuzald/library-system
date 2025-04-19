package service

import (
	"context"
	"errors"
	"strings"

	"github.com/fairuzald/library-system/pkg/constants"
	"github.com/fairuzald/library-system/pkg/logger"
	"github.com/fairuzald/library-system/pkg/utils"
	"github.com/fairuzald/library-system/services/user-service/internal/entity/dao"
	"github.com/fairuzald/library-system/services/user-service/internal/entity/dto"
	"github.com/fairuzald/library-system/services/user-service/internal/entity/model"
	"github.com/fairuzald/library-system/services/user-service/internal/repository"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type userService struct {
	userRepo repository.UserRepository
	log      *logger.Logger
}

func NewUserService(userRepo repository.UserRepository, log *logger.Logger) UserService {
	return &userService{
		userRepo: userRepo,
		log:      log,
	}
}

func (s *userService) GetUserByID(ctx context.Context, id uuid.UUID) (*dao.UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), constants.ErrUserNotFound) {
			return nil, errors.New(constants.ErrUserNotFound)
		}
		s.log.Error("Failed to get user by ID", zap.Error(err), zap.String("id", id.String()))
		return nil, errors.New(constants.ErrInternalServer)
	}

	return dao.NewUserResponse(user), nil
}

func (s *userService) GetUserByEmail(ctx context.Context, email string) (*dao.UserResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		s.log.Error("Failed to get user by email", zap.Error(err), zap.String("email", email))
		return nil, err
	}

	return dao.NewUserResponse(user), nil
}

func (s *userService) GetUserByUsername(ctx context.Context, username string) (*dao.UserResponse, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		s.log.Error("Failed to get user by username", zap.Error(err), zap.String("username", username))
		return nil, err
	}

	return dao.NewUserResponse(user), nil
}

func (s *userService) UpdateUser(ctx context.Context, id uuid.UUID, req *dto.UserUpdate) (*dao.UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Email != nil && *req.Email != user.Email {
		existingUser, err := s.userRepo.GetByEmail(ctx, *req.Email)
		if err == nil && existingUser.ID != id {
			return nil, errors.New(constants.ErrEmailTaken)
		}
	}

	if req.Username != nil && *req.Username != user.Username {
		existingUser, err := s.userRepo.GetByUsername(ctx, *req.Username)
		if err == nil && existingUser.ID != id {
			return nil, errors.New(constants.ErrUsernameTaken)
		}
	}

	if req.Email != nil {
		user.Email = *req.Email
	}

	if req.Username != nil {
		user.Username = *req.Username
	}

	if req.FirstName != nil {
		user.FirstName = *req.FirstName
	}

	if req.LastName != nil {
		user.LastName = *req.LastName
	}

	if req.Role != nil {
		user.Role = *req.Role
	}

	if req.Status != nil {
		user.Status = *req.Status
	}

	if req.Phone != nil {
		user.Phone = *req.Phone
	}

	if req.Address != nil {
		user.Address = *req.Address
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		s.log.Error("Failed to update user", zap.Error(err), zap.String("id", id.String()))
		return nil, err
	}

	return dao.NewUserResponse(user), nil
}

func (s *userService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	_, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	return s.userRepo.Delete(ctx, id)
}

func (s *userService) ListUsers(ctx context.Context, filter *dto.UserFilter) (*dao.UserListResponse, error) {
	filter.Validate()

	users, count, err := s.userRepo.List(ctx, filter)
	if err != nil {
		s.log.Error("Failed to list users", zap.Error(err))
		return nil, err
	}

	response := &dao.UserListResponse{
		Users:       make([]dao.UserResponse, 0, len(users)),
		TotalItems:  count,
		TotalPages:  (int(count) + filter.Limit - 1) / filter.Limit,
		CurrentPage: filter.Page,
		PageSize:    filter.Limit,
	}

	for _, user := range users {
		response.Users = append(response.Users, *dao.NewUserResponse(user))
	}

	return response, nil
}

func (s *userService) ChangePassword(ctx context.Context, id uuid.UUID, req *dto.ChangePassword) error {
	if req.CurrentPassword == "" || req.NewPassword == "" {
		return errors.New(constants.ErrRequiredField)
	}

	if len(req.NewPassword) < constants.PasswordMinLength {
		return errors.New(constants.ErrWeakPassword)
	}

	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if !utils.CheckPasswordHash(req.CurrentPassword, user.Password) {
		return errors.New(constants.ErrInvalidCredentials)
	}

	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		s.log.Error("Failed to hash password", zap.Error(err))
		return err
	}

	user.Password = hashedPassword

	return s.userRepo.Update(ctx, user)
}

func (s *userService) CreateUser(ctx context.Context, req *dto.UserCreate) (*dao.UserResponse, error) {
	if req.Email == "" || req.Username == "" || req.Password == "" {
		return nil, errors.New(constants.ErrRequiredField)
	}

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
		req.Role,
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
