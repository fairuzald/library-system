package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/fairuzald/library-system/pkg/constants"
	"github.com/fairuzald/library-system/pkg/logger"
	"github.com/fairuzald/library-system/services/category-service/internal/entity/dao"
	"github.com/fairuzald/library-system/services/category-service/internal/entity/dto"
	"github.com/fairuzald/library-system/services/category-service/internal/entity/model"
	"github.com/fairuzald/library-system/services/category-service/internal/repository"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type categoryService struct {
	categoryRepo repository.CategoryRepository
	log          *logger.Logger
}

func NewCategoryService(categoryRepo repository.CategoryRepository, log *logger.Logger) CategoryService {
	return &categoryService{
		categoryRepo: categoryRepo,
		log:          log,
	}
}

func (s *categoryService) CreateCategory(ctx context.Context, req *dto.CategoryCreate) (*dao.CategoryResponse, error) {
	existingCategory, err := s.categoryRepo.GetByName(ctx, req.Name)
	if err == nil && existingCategory != nil {
		return nil, fmt.Errorf("category with name %s already exists", req.Name)
	}

	var parentID *uuid.UUID
	if req.ParentID != nil && *req.ParentID != "" {
		id, err := uuid.Parse(*req.ParentID)
		if err != nil {
			return nil, fmt.Errorf("invalid parent ID format: %s", *req.ParentID)
		}

		parent, err := s.categoryRepo.GetByID(ctx, id)
		if err != nil {
			if strings.Contains(err.Error(), constants.ErrCategoryNotFound) {
				return nil, fmt.Errorf("parent category not found")
			}
			s.log.Error("Failed to get parent category", zap.Error(err), zap.String("parent_id", id.String()))
			return nil, errors.New(constants.ErrInternalServer)
		}

		parentID = &parent.ID
	}

	category := model.NewCategory(req.Name, req.Description, parentID)

	if err := s.categoryRepo.Create(ctx, category); err != nil {
		s.log.Error("Failed to create category", zap.Error(err))
		return nil, errors.New(constants.ErrInternalServer)
	}

	return dao.NewCategoryResponse(category), nil
}

func (s *categoryService) GetCategoryByID(ctx context.Context, id uuid.UUID) (*dao.CategoryResponse, error) {
	category, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), constants.ErrCategoryNotFound) {
			return nil, errors.New(constants.ErrCategoryNotFound)
		}
		s.log.Error("Failed to get category", zap.Error(err), zap.String("id", id.String()))
		return nil, errors.New(constants.ErrInternalServer)
	}

	return dao.NewCategoryResponse(category), nil
}

func (s *categoryService) GetCategoryByName(ctx context.Context, name string) (*dao.CategoryResponse, error) {
	category, err := s.categoryRepo.GetByName(ctx, name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, fmt.Errorf("category with name %s not found", name)
		}
		s.log.Error("Failed to get category by name", zap.Error(err), zap.String("name", name))
		return nil, errors.New(constants.ErrInternalServer)
	}

	return dao.NewCategoryResponse(category), nil
}

func (s *categoryService) UpdateCategory(ctx context.Context, id uuid.UUID, req *dto.CategoryUpdate) (*dao.CategoryResponse, error) {
	category, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), constants.ErrCategoryNotFound) {
			return nil, errors.New(constants.ErrCategoryNotFound)
		}
		s.log.Error("Failed to get category for update", zap.Error(err), zap.String("id", id.String()))
		return nil, errors.New(constants.ErrInternalServer)
	}

	if req.Name != nil && *req.Name != category.Name {
		existingCategory, err := s.categoryRepo.GetByName(ctx, *req.Name)
		if err == nil && existingCategory != nil && existingCategory.ID != id {
			return nil, fmt.Errorf("category with name %s already exists", *req.Name)
		}
	}

	if req.Name != nil {
		category.Name = *req.Name
	}

	if req.Description != nil {
		category.Description = *req.Description
	}

	if req.ParentID != nil {
		if *req.ParentID == "" {
			category.ParentID = nil
		} else {
			parentID, err := uuid.Parse(*req.ParentID)
			if err != nil {
				return nil, fmt.Errorf("invalid parent ID format: %s", *req.ParentID)
			}

			if parentID == id {
				return nil, errors.New("category cannot be its own parent")
			}

			parent, err := s.categoryRepo.GetByID(ctx, parentID)
			if err != nil {
				if strings.Contains(err.Error(), constants.ErrCategoryNotFound) {
					return nil, fmt.Errorf("parent category not found")
				}
				s.log.Error("Failed to get parent category", zap.Error(err), zap.String("parent_id", parentID.String()))
				return nil, errors.New(constants.ErrInternalServer)
			}

			if err := s.ensureNoCycle(ctx, parentID, id); err != nil {
				return nil, err
			}

			category.ParentID = &parent.ID
		}
	}

	if err := s.categoryRepo.Update(ctx, category); err != nil {
		s.log.Error("Failed to update category", zap.Error(err), zap.String("id", id.String()))
		return nil, errors.New(constants.ErrInternalServer)
	}

	return dao.NewCategoryResponse(category), nil
}

func (s *categoryService) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	_, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), constants.ErrCategoryNotFound) {
			return errors.New(constants.ErrCategoryNotFound)
		}
		s.log.Error("Failed to get category for deletion", zap.Error(err), zap.String("id", id.String()))
		return errors.New(constants.ErrInternalServer)
	}

	if err := s.categoryRepo.Delete(ctx, id); err != nil {
		if strings.Contains(err.Error(), "child categories") {
			return errors.New("cannot delete category with child categories")
		}
		if strings.Contains(err.Error(), "associated books") {
			return errors.New("cannot delete category with associated books")
		}
		s.log.Error("Failed to delete category", zap.Error(err), zap.String("id", id.String()))
		return errors.New(constants.ErrInternalServer)
	}

	return nil
}

func (s *categoryService) ListCategories(ctx context.Context, filter *dto.CategoryFilter) (*dao.CategoryListResponse, error) {
	filter.Validate()

	categories, count, err := s.categoryRepo.List(ctx, filter)
	if err != nil {
		s.log.Error("Failed to list categories", zap.Error(err))
		return nil, errors.New(constants.ErrInternalServer)
	}

	response := &dao.CategoryListResponse{
		Categories:  make([]dao.CategoryResponse, 0, len(categories)),
		TotalItems:  count,
		TotalPages:  (int(count) + filter.Limit - 1) / filter.Limit,
		CurrentPage: filter.Page,
		PageSize:    filter.Limit,
	}

	for _, category := range categories {
		response.Categories = append(response.Categories, *dao.NewCategoryResponse(category))
	}

	return response, nil
}

func (s *categoryService) GetCategoryChildren(ctx context.Context, parentID uuid.UUID) (*dao.CategoryListResponse, error) {
	_, err := s.categoryRepo.GetByID(ctx, parentID)
	if err != nil {
		if strings.Contains(err.Error(), constants.ErrCategoryNotFound) {
			return nil, errors.New(constants.ErrCategoryNotFound)
		}
		s.log.Error("Failed to get parent category", zap.Error(err), zap.String("parent_id", parentID.String()))
		return nil, errors.New(constants.ErrInternalServer)
	}

	children, err := s.categoryRepo.GetChildren(ctx, parentID)
	if err != nil {
		s.log.Error("Failed to get category children", zap.Error(err), zap.String("parent_id", parentID.String()))
		return nil, errors.New(constants.ErrInternalServer)
	}

	response := &dao.CategoryListResponse{
		Categories:  make([]dao.CategoryResponse, 0, len(children)),
		TotalItems:  int64(len(children)),
		TotalPages:  1,
		CurrentPage: 1,
		PageSize:    len(children),
	}

	for _, child := range children {
		response.Categories = append(response.Categories, *dao.NewCategoryResponse(child))
	}

	return response, nil
}

func (s *categoryService) ensureNoCycle(ctx context.Context, parentID, childID uuid.UUID) error {
	visited := make(map[uuid.UUID]bool)

	var checkCycle func(current uuid.UUID) error
	checkCycle = func(current uuid.UUID) error {
		if current == childID {
			return errors.New("operation would create a category cycle")
		}

		if visited[current] {
			return nil
		}

		visited[current] = true

		category, err := s.categoryRepo.GetByID(ctx, current)
		if err != nil {
			return err
		}

		if category.ParentID != nil {
			return checkCycle(*category.ParentID)
		}

		return nil
	}

	return checkCycle(parentID)
}
