package model

import (
	"time"

	"github.com/fairuzald/library-system/pkg/constants"
	"github.com/fairuzald/library-system/pkg/models"
	"github.com/google/uuid"
)

type User struct {
	models.Base
	Email           string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	Username        string    `gorm:"type:varchar(30);uniqueIndex;not null" json:"username"`
	Password        string    `gorm:"type:varchar(255);not null" json:"-"`
	FirstName       string    `gorm:"type:varchar(100);not null" json:"first_name"`
	LastName        string    `gorm:"type:varchar(100);not null" json:"last_name"`
	Role            string    `gorm:"type:varchar(20);not null;default:'member'" json:"role"`
	Status          string    `gorm:"type:varchar(20);not null;default:'active'" json:"status"`
	Phone           string    `gorm:"type:varchar(20)" json:"phone,omitempty"`
	Address         string    `gorm:"type:text" json:"address,omitempty"`
	LastLogin       time.Time `gorm:"type:timestamp" json:"last_login,omitempty"`
	RefreshToken    string    `gorm:"type:varchar(255)" json:"-"`
	RefreshTokenExp time.Time `gorm:"type:timestamp" json:"-"`
}

func (User) TableName() string {
	return "users"
}

func NewUser(email, username, password, firstName, lastName, role string) *User {
	if role == "" {
		role = constants.RoleMember
	}

	return &User{
		Base: models.Base{
			ID: uuid.New(),
		},
		Email:     email,
		Username:  username,
		Password:  password,
		FirstName: firstName,
		LastName:  lastName,
		Role:      role,
		Status:    constants.UserStatusActive,
	}
}
