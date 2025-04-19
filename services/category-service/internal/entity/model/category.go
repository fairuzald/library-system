package model

import (
	"time"

	"github.com/fairuzald/library-system/pkg/models"
	"github.com/google/uuid"
)

type Category struct {
	models.Base
	Name        string     `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
	Description string     `gorm:"type:text" json:"description"`
	ParentID    *uuid.UUID `gorm:"type:uuid" json:"parent_id,omitempty"`
}

func (Category) TableName() string {
	return "categories"
}

func NewCategory(name, description string, parentID *uuid.UUID) *Category {
	return &Category{
		Base: models.Base{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Name:        name,
		Description: description,
		ParentID:    parentID,
	}
}
