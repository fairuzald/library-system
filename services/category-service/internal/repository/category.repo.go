package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/fairuzald/library-system/pkg/cache"
	"github.com/fairuzald/library-system/pkg/constants"
	"github.com/fairuzald/library-system/pkg/logger"
	"github.com/fairuzald/library-system/services/category-service/internal/entity/dto"
	"github.com/fairuzald/library-system/services/category-service/internal/entity/model"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CategoryRepository interface {
	Create(ctx context.Context, category *model.Category) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Category, error)
	GetByName(ctx context.Context, name string) (*model.Category, error)
	Update(ctx context.Context, category *model.Category) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter *dto.CategoryFilter) ([]*model.Category, int64, error)
	GetChildren(ctx context.Context, parentID uuid.UUID) ([]*model.Category, error)
	HasBooks(ctx context.Context, id uuid.UUID) (bool, error)
}

type categoryRepository struct {
	db    *gorm.DB
	cache *cache.Redis
	log   *logger.Logger
}

func NewCategoryRepository(db *gorm.DB, cache *cache.Redis, log *logger.Logger) CategoryRepository {
	return &categoryRepository{
		db:    db,
		cache: cache,
		log:   log,
	}
}

func (r *categoryRepository) Create(ctx context.Context, category *model.Category) error {
	err := r.db.WithContext(ctx).Create(category).Error
	if err != nil {
		r.log.Error("Failed to create category", zap.Error(err), zap.String("name", category.Name))
		return err
	}

	if r.cache != nil {
		cacheKey := fmt.Sprintf("%slist", constants.CacheKeyCategories)
		_ = r.cache.Delete(ctx, cacheKey)
	}

	return nil
}

func (r *categoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Category, error) {
	var category model.Category

	if r.cache != nil {
		cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyCategory, id.String())
		err := r.cache.Get(ctx, cacheKey, &category)
		if err == nil {
			return &category, nil
		}
	}

	err := r.db.WithContext(ctx).Where("id = ?", id).First(&category).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%s: %w", constants.ErrCategoryNotFound, err)
		}
		return nil, err
	}

	if r.cache != nil {
		cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyCategory, id.String())
		_ = r.cache.Set(ctx, cacheKey, category, constants.CacheDefaultTTL)
	}

	return &category, nil
}

func (r *categoryRepository) GetByName(ctx context.Context, name string) (*model.Category, error) {
	var category model.Category

	if r.cache != nil {
		cacheKey := fmt.Sprintf("%sname:%s", constants.CacheKeyCategory, name)
		err := r.cache.Get(ctx, cacheKey, &category)
		if err == nil {
			return &category, nil
		}
	}

	err := r.db.WithContext(ctx).Where("name = ?", name).First(&category).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("category with name %s not found: %w", name, err)
		}
		return nil, err
	}

	if r.cache != nil {
		cacheKey := fmt.Sprintf("%sname:%s", constants.CacheKeyCategory, name)
		_ = r.cache.Set(ctx, cacheKey, category, constants.CacheDefaultTTL)
	}

	return &category, nil
}

func (r *categoryRepository) Update(ctx context.Context, category *model.Category) error {
	err := r.db.WithContext(ctx).Save(category).Error
	if err != nil {
		r.log.Error("Failed to update category", zap.Error(err), zap.String("id", category.ID.String()))
		return err
	}

	if r.cache != nil {
		// Clear cache for this category
		cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyCategory, category.ID.String())
		_ = r.cache.Delete(ctx, cacheKey)
		// Clear cache for category by name
		cacheKey = fmt.Sprintf("%sname:%s", constants.CacheKeyCategory, category.Name)
		_ = r.cache.Delete(ctx, cacheKey)
		// Clear list cache
		cacheKey = fmt.Sprintf("%slist", constants.CacheKeyCategories)
		_ = r.cache.Delete(ctx, cacheKey)
	}

	return nil
}

func (r *categoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	category, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Check if there are any child categories
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.Category{}).Where("parent_id = ?", id).Count(&count).Error; err != nil {
		r.log.Error("Failed to count child categories", zap.Error(err), zap.String("id", id.String()))
		return err
	}

	if count > 0 {
		return fmt.Errorf("cannot delete category with child categories")
	}

	// Check if there are any books associated with this category
	hasBooks, err := r.HasBooks(ctx, id)
	if err != nil {
		return err
	}

	if hasBooks {
		return fmt.Errorf("cannot delete category with associated books")
	}

	if err := r.db.WithContext(ctx).Delete(&model.Category{}, id).Error; err != nil {
		r.log.Error("Failed to delete category", zap.Error(err), zap.String("id", id.String()))
		return err
	}

	if r.cache != nil {
		cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyCategory, id.String())
		_ = r.cache.Delete(ctx, cacheKey)
		cacheKey = fmt.Sprintf("%sname:%s", constants.CacheKeyCategory, category.Name)
		_ = r.cache.Delete(ctx, cacheKey)
		cacheKey = fmt.Sprintf("%slist", constants.CacheKeyCategories)
		_ = r.cache.Delete(ctx, cacheKey)
	}

	return nil
}

func (r *categoryRepository) List(ctx context.Context, filter *dto.CategoryFilter) ([]*model.Category, int64, error) {
	var categories []*model.Category
	var count int64

	query := r.db.WithContext(ctx).Model(&model.Category{})

	if filter.ParentID != nil {
		if *filter.ParentID == "null" {
			query = query.Where("parent_id IS NULL")
		} else {
			parentID, err := uuid.Parse(*filter.ParentID)
			if err == nil {
				query = query.Where("parent_id = ?", parentID)
			}
		}
	}

	if err := query.Count(&count).Error; err != nil {
		r.log.Error("Failed to count categories", zap.Error(err))
		return nil, 0, err
	}

	if filter.SortBy != "" {
		if filter.Desc {
			query = query.Order(fmt.Sprintf("%s DESC", filter.SortBy))
		} else {
			query = query.Order(filter.SortBy)
		}
	} else {
		query = query.Order("name ASC")
	}

	query = query.Offset(filter.GetOffset()).Limit(filter.Limit)

	if err := query.Find(&categories).Error; err != nil {
		r.log.Error("Failed to list categories", zap.Error(err))
		return nil, 0, err
	}

	return categories, count, nil
}

func (r *categoryRepository) GetChildren(ctx context.Context, parentID uuid.UUID) ([]*model.Category, error) {
	var categories []*model.Category

	err := r.db.WithContext(ctx).Where("parent_id = ?", parentID).Find(&categories).Error
	if err != nil {
		r.log.Error("Failed to get child categories", zap.Error(err), zap.String("parent_id", parentID.String()))
		return nil, err
	}

	return categories, nil
}

func (r *categoryRepository) HasBooks(ctx context.Context, id uuid.UUID) (bool, error) {
	// This would typically check in the books_categories table in the book service
	// However, since we're in a microservice architecture, this has to be handled differently
	// For now, we'll assume there's a local books_categories table we can check

	var count int64
	err := r.db.WithContext(ctx).Table("books_categories").Where("category_id = ?", id).Count(&count).Error
	if err != nil {
		// If the table doesn't exist, we assume there are no books
		if errors.Is(err, gorm.ErrUnsupportedRelation) {
			return false, nil
		}
		r.log.Error("Failed to check if category has books", zap.Error(err), zap.String("id", id.String()))
		return false, err
	}

	return count > 0, nil
}
