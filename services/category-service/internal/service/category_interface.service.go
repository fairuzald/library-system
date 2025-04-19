package service

import (
	"context"

	"github.com/fairuzald/library-system/services/category-service/internal/entity/dao"
	"github.com/fairuzald/library-system/services/category-service/internal/entity/dto"
	"github.com/google/uuid"
)

type CategoryService interface {
	CreateCategory(ctx context.Context, req *dto.CategoryCreate) (*dao.CategoryResponse, error)
	GetCategoryByID(ctx context.Context, id uuid.UUID) (*dao.CategoryResponse, error)
	GetCategoryByName(ctx context.Context, name string) (*dao.CategoryResponse, error)
	UpdateCategory(ctx context.Context, id uuid.UUID, req *dto.CategoryUpdate) (*dao.CategoryResponse, error)
	DeleteCategory(ctx context.Context, id uuid.UUID) error
	ListCategories(ctx context.Context, filter *dto.CategoryFilter) (*dao.CategoryListResponse, error)
	GetCategoryChildren(ctx context.Context, parentID uuid.UUID) (*dao.CategoryListResponse, error)
}
