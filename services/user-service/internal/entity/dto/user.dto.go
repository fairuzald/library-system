package dto

import (
	"github.com/fairuzald/library-system/pkg/constants"
)

type UserCreate struct {
	Email     string `json:"email" validate:"required,email"`
	Username  string `json:"username" validate:"required,min=3,max=30"`
	Password  string `json:"password" validate:"required,min=8"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
	Role      string `json:"role" validate:"required,oneof=admin librarian member guest"`
	Phone     string `json:"phone,omitempty"`
	Address   string `json:"address,omitempty"`
}

type UserUpdate struct {
	Email     *string `json:"email,omitempty" validate:"omitempty,email"`
	Username  *string `json:"username,omitempty" validate:"omitempty,min=3,max=30"`
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	Role      *string `json:"role,omitempty" validate:"omitempty,oneof=admin librarian member guest"`
	Status    *string `json:"status,omitempty" validate:"omitempty,oneof=active inactive pending blocked"`
	Phone     *string `json:"phone,omitempty"`
	Address   *string `json:"address,omitempty"`
}

type UserLogin struct {
	UsernameOrEmail string `json:"username_or_email" validate:"required"`
	Password        string `json:"password" validate:"required"`
}

type ChangePassword struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
}

type RefreshToken struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type UserFilter struct {
	Role   string `form:"role" query:"role"`
	Status string `form:"status" query:"status"`
	Page   int    `form:"page,default=1" query:"page,default=1"`
	Limit  int    `form:"limit,default=10" query:"limit,default=10"`
	SortBy string `form:"sort_by,default=created_at" query:"sort_by,default=created_at"`
	Desc   bool   `form:"desc" query:"desc"`
}

type UserRegister struct {
	Email     string `json:"email" validate:"required,email"`
	Username  string `json:"username" validate:"required,min=3,max=30"`
	Password  string `json:"password" validate:"required,min=8"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
	Phone     string `json:"phone,omitempty"`
	Address   string `json:"address,omitempty"`
}

func (f *UserFilter) Validate() {
	if f.Page <= 0 {
		f.Page = 1
	}

	if f.Limit <= 0 {
		f.Limit = constants.DefaultPageSize
	} else if f.Limit > constants.MaxPageSize {
		f.Limit = constants.MaxPageSize
	}

	if f.Role != "" && f.Role != constants.RoleAdmin && f.Role != constants.RoleLibrarian &&
		f.Role != constants.RoleMember && f.Role != constants.RoleGuest {
		f.Role = ""
	}

	if f.Status != "" && f.Status != constants.UserStatusActive && f.Status != constants.UserStatusInactive &&
		f.Status != constants.UserStatusPending && f.Status != constants.UserStatusBlocked {
		f.Status = ""
	}
}

func (f *UserFilter) GetOffset() int {
	return (f.Page - 1) * f.Limit
}
