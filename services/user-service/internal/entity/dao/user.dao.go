package dao

import (
	"time"

	"github.com/fairuzald/library-system/services/user-service/internal/entity/model"
	"github.com/google/uuid"
)

type UserResponse struct {
	ID        uuid.UUID  `json:"id"`
	Email     string     `json:"email"`
	Username  string     `json:"username"`
	FirstName string     `json:"first_name"`
	LastName  string     `json:"last_name"`
	Role      string     `json:"role"`
	Status    string     `json:"status"`
	Phone     string     `json:"phone,omitempty"`
	Address   string     `json:"address,omitempty"`
	LastLogin *time.Time `json:"last_login,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func NewUserResponse(user *model.User) *UserResponse {
	response := &UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Username:  user.Username,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Role:      user.Role,
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	if user.Phone != "" {
		response.Phone = user.Phone
	}

	if user.Address != "" {
		response.Address = user.Address
	}

	if !user.LastLogin.IsZero() {
		lastLogin := user.LastLogin
		response.LastLogin = &lastLogin
	}

	return response
}

type UserListResponse struct {
	Users       []UserResponse `json:"users"`
	TotalItems  int64          `json:"total_items"`
	TotalPages  int            `json:"total_pages"`
	CurrentPage int            `json:"current_page"`
	PageSize    int            `json:"page_size"`
}

type TokenResponse struct {
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token,omitempty"`
	TokenType    string        `json:"token_type"`
	ExpiresIn    int64         `json:"expires_in"` // in seconds
	User         *UserResponse `json:"user,omitempty"`
}
