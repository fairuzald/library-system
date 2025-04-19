package dao

import (
	"time"

	"github.com/fairuzald/library-system/services/category-service/internal/entity/model"
	"github.com/google/uuid"
)

type CategoryResponse struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func NewCategoryResponse(category *model.Category) *CategoryResponse {
	return &CategoryResponse{
		ID:          category.ID,
		Name:        category.Name,
		Description: category.Description,
		ParentID:    category.ParentID,
		CreatedAt:   category.CreatedAt,
		UpdatedAt:   category.UpdatedAt,
	}
}

type CategoryListResponse struct {
	Categories  []CategoryResponse `json:"categories"`
	TotalItems  int64              `json:"total_items"`
	TotalPages  int                `json:"total_pages"`
	CurrentPage int                `json:"current_page"`
	PageSize    int                `json:"page_size"`
}
