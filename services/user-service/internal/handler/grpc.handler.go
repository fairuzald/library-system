package handler

import (
	"context"
	"strings"

	"github.com/fairuzald/library-system/pkg/constants"
	"github.com/fairuzald/library-system/pkg/logger"
	"github.com/fairuzald/library-system/proto/user"
	"github.com/fairuzald/library-system/services/user-service/internal/entity/dao"
	"github.com/fairuzald/library-system/services/user-service/internal/entity/dto"
	"github.com/fairuzald/library-system/services/user-service/internal/service"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type UserService struct {
	user.UnimplementedUserServiceServer
	userService service.UserService
	authService service.AuthService
	log         *logger.Logger
}

func NewUserService(userService service.UserService, authService service.AuthService, log *logger.Logger) *UserService {
	return &UserService{
		userService: userService,
		authService: authService,
		log:         log,
	}
}

func (s *UserService) GetUser(ctx context.Context, req *user.GetUserRequest) (*user.UserResponse, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	userResponse, err := s.userService.GetUserByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), constants.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, constants.ErrUserNotFound)
		}
		s.log.Error("Failed to get user", zap.Error(err), zap.String("id", id.String()))
		return nil, status.Error(codes.Internal, constants.ErrInternalServer)
	}

	return convertUserToProto(userResponse), nil
}

func (s *UserService) GetUserByEmail(ctx context.Context, req *user.GetUserByEmailRequest) (*user.UserResponse, error) {
	if req.GetEmail() == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	userResponse, err := s.userService.GetUserByEmail(ctx, req.GetEmail())
	if err != nil {
		if strings.Contains(err.Error(), constants.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, constants.ErrUserNotFound)
		}
		s.log.Error("Failed to get user by email", zap.Error(err), zap.String("email", req.GetEmail()))
		return nil, status.Error(codes.Internal, constants.ErrInternalServer)
	}

	return convertUserToProto(userResponse), nil
}

func (s *UserService) GetUserByUsername(ctx context.Context, req *user.GetUserByUsernameRequest) (*user.UserResponse, error) {
	if req.GetUsername() == "" {
		return nil, status.Error(codes.InvalidArgument, "username is required")
	}

	userResponse, err := s.userService.GetUserByUsername(ctx, req.GetUsername())
	if err != nil {
		if strings.Contains(err.Error(), constants.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, constants.ErrUserNotFound)
		}
		s.log.Error("Failed to get user by username", zap.Error(err), zap.String("username", req.GetUsername()))
		return nil, status.Error(codes.Internal, constants.ErrInternalServer)
	}

	return convertUserToProto(userResponse), nil
}

func (s *UserService) ListUsers(ctx context.Context, req *user.ListUsersRequest) (*user.ListUsersResponse, error) {
	filter := &dto.UserFilter{
		Page:  int(req.GetPage()),
		Limit: int(req.GetPageSize()),
	}

	if req.Role != nil {
		filter.Role = req.GetRole()
	}

	if req.Status != nil {
		filter.Status = req.GetStatus()
	}

	if req.SortBy != nil {
		filter.SortBy = req.GetSortBy()
	}

	if req.SortDesc != nil {
		filter.Desc = req.GetSortDesc()
	}

	response, err := s.userService.ListUsers(ctx, filter)
	if err != nil {
		s.log.Error("Failed to list users", zap.Error(err))
		return nil, status.Error(codes.Internal, constants.ErrInternalServer)
	}

	protoResponse := &user.ListUsersResponse{
		Users:       make([]*user.User, 0, len(response.Users)),
		TotalItems:  int64(response.TotalItems),
		TotalPages:  int32(response.TotalPages),
		CurrentPage: int32(response.CurrentPage),
		PageSize:    int32(response.PageSize),
	}

	for _, u := range response.Users {
		protoResponse.Users = append(protoResponse.Users, convertDaoUserToProtoUser(&u))
	}

	return protoResponse, nil
}

func (s *UserService) CreateUser(ctx context.Context, req *user.CreateUserRequest) (*user.UserResponse, error) {
	createDTO := &dto.UserCreate{
		Email:     req.GetEmail(),
		Username:  req.GetUsername(),
		Password:  req.GetPassword(),
		FirstName: req.GetFirstName(),
		LastName:  req.GetLastName(),
		Role:      req.GetRole(),
	}

	if req.Phone != nil {
		createDTO.Phone = req.GetPhone()
	}

	if req.Address != nil {
		createDTO.Address = req.GetAddress()
	}

	userResponse, err := s.userService.CreateUser(ctx, createDTO)
	if err != nil {
		if err.Error() == constants.ErrEmailTaken || err.Error() == constants.ErrUsernameTaken {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		s.log.Error("Failed to create user", zap.Error(err))
		return nil, status.Error(codes.Internal, constants.ErrInternalServer)
	}

	return convertUserToProto(userResponse), nil
}

func (s *UserService) UpdateUser(ctx context.Context, req *user.UpdateUserRequest) (*user.UserResponse, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	updateDTO := &dto.UserUpdate{}

	if req.Email != nil {
		email := req.GetEmail()
		updateDTO.Email = &email
	}

	if req.Username != nil {
		username := req.GetUsername()
		updateDTO.Username = &username
	}

	if req.FirstName != nil {
		firstName := req.GetFirstName()
		updateDTO.FirstName = &firstName
	}

	if req.LastName != nil {
		lastName := req.GetLastName()
		updateDTO.LastName = &lastName
	}

	if req.Role != nil {
		role := req.GetRole()
		updateDTO.Role = &role
	}

	if req.Status != nil {
		status := req.GetStatus()
		updateDTO.Status = &status
	}

	if req.Phone != nil {
		phone := req.GetPhone()
		updateDTO.Phone = &phone
	}

	if req.Address != nil {
		address := req.GetAddress()
		updateDTO.Address = &address
	}

	userResponse, err := s.userService.UpdateUser(ctx, id, updateDTO)
	if err != nil {
		if strings.Contains(err.Error(), constants.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, constants.ErrUserNotFound)
		}
		if err.Error() == constants.ErrEmailTaken || err.Error() == constants.ErrUsernameTaken {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		s.log.Error("Failed to update user", zap.Error(err), zap.String("id", id.String()))
		return nil, status.Error(codes.Internal, constants.ErrInternalServer)
	}

	return convertUserToProto(userResponse), nil
}

func (s *UserService) DeleteUser(ctx context.Context, req *user.DeleteUserRequest) (*emptypb.Empty, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	if err := s.userService.DeleteUser(ctx, id); err != nil {
		if strings.Contains(err.Error(), constants.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, constants.ErrUserNotFound)
		}
		s.log.Error("Failed to delete user", zap.Error(err), zap.String("id", id.String()))
		return nil, status.Error(codes.Internal, constants.ErrInternalServer)
	}

	return &emptypb.Empty{}, nil
}

func (s *UserService) ChangePassword(ctx context.Context, req *user.ChangePasswordRequest) (*emptypb.Empty, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	changePasswordDTO := &dto.ChangePassword{
		CurrentPassword: req.GetCurrentPassword(),
		NewPassword:     req.GetNewPassword(),
	}

	if err := s.userService.ChangePassword(ctx, id, changePasswordDTO); err != nil {
		if strings.Contains(err.Error(), constants.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, constants.ErrUserNotFound)
		}
		if err.Error() == constants.ErrInvalidCredentials {
			return nil, status.Error(codes.Unauthenticated, constants.ErrInvalidCredentials)
		}
		if err.Error() == constants.ErrWeakPassword {
			return nil, status.Error(codes.InvalidArgument, constants.ErrWeakPassword)
		}
		s.log.Error("Failed to change password", zap.Error(err), zap.String("id", id.String()))
		return nil, status.Error(codes.Internal, constants.ErrInternalServer)
	}

	return &emptypb.Empty{}, nil
}

func (s *UserService) Login(ctx context.Context, req *user.LoginRequest) (*user.LoginResponse, error) {
	loginDTO := &dto.UserLogin{
		UsernameOrEmail: req.GetUsernameOrEmail(),
		Password:        req.GetPassword(),
	}

	tokenResponse, err := s.authService.Login(ctx, loginDTO)
	if err != nil {
		if err.Error() == constants.ErrInvalidCredentials {
			return nil, status.Error(codes.Unauthenticated, constants.ErrInvalidCredentials)
		}
		s.log.Error("Login failed", zap.Error(err))
		return nil, status.Error(codes.Internal, constants.ErrInternalServer)
	}

	response := &user.LoginResponse{
		AccessToken:  tokenResponse.AccessToken,
		RefreshToken: tokenResponse.RefreshToken,
		TokenType:    tokenResponse.TokenType,
		ExpiresIn:    tokenResponse.ExpiresIn,
		User:         convertDaoUserToProtoUser(tokenResponse.User),
	}

	return response, nil
}

func (s *UserService) RefreshToken(ctx context.Context, req *user.RefreshTokenRequest) (*user.TokenResponse, error) {
	tokenResponse, err := s.authService.RefreshToken(ctx, req.GetRefreshToken())
	if err != nil {
		if err.Error() == constants.ErrInvalidToken {
			return nil, status.Error(codes.Unauthenticated, constants.ErrInvalidToken)
		}
		s.log.Error("Token refresh failed", zap.Error(err))
		return nil, status.Error(codes.Internal, constants.ErrInternalServer)
	}

	response := &user.TokenResponse{
		AccessToken:  tokenResponse.AccessToken,
		RefreshToken: tokenResponse.RefreshToken,
		TokenType:    tokenResponse.TokenType,
		ExpiresIn:    tokenResponse.ExpiresIn,
	}

	return response, nil
}

func (s *UserService) Logout(ctx context.Context, req *user.LogoutRequest) (*emptypb.Empty, error) {
	if err := s.authService.Logout(ctx, req.GetRefreshToken()); err != nil {
		if err.Error() == constants.ErrInvalidToken {
			return nil, status.Error(codes.Unauthenticated, constants.ErrInvalidToken)
		}
		s.log.Error("Logout failed", zap.Error(err))
		return nil, status.Error(codes.Internal, constants.ErrInternalServer)
	}

	return &emptypb.Empty{}, nil
}

func (s *UserService) Health(ctx context.Context, _ *emptypb.Empty) (*user.HealthResponse, error) {
	return &user.HealthResponse{
		Status:  "ok",
		Version: "1.0.0",
	}, nil
}

func convertUserToProto(u *dao.UserResponse) *user.UserResponse {
	return &user.UserResponse{
		User: convertDaoUserToProtoUser(u),
	}
}

func convertDaoUserToProtoUser(u *dao.UserResponse) *user.User {
	protoUser := &user.User{
		Id:        u.ID.String(),
		Email:     u.Email,
		Username:  u.Username,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Role:      u.Role,
		Status:    u.Status,
		CreatedAt: timestamppb.New(u.CreatedAt),
		UpdatedAt: timestamppb.New(u.UpdatedAt),
	}

	if u.Phone != "" {
		protoUser.Phone = &u.Phone
	}

	if u.Address != "" {
		protoUser.Address = &u.Address
	}

	if u.LastLogin != nil {
		protoUser.LastLogin = timestamppb.New(*u.LastLogin)
	}

	return protoUser
}
