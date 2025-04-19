package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fairuzald/library-system/pkg/cache"
	"github.com/fairuzald/library-system/pkg/constants"
	"github.com/fairuzald/library-system/pkg/logger"
	"github.com/fairuzald/library-system/services/user-service/internal/entity/dto"
	"github.com/fairuzald/library-system/services/user-service/internal/entity/model"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByUsernameOrEmail(ctx context.Context, usernameOrEmail string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter *dto.UserFilter) ([]*model.User, int64, error)
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
}

type userRepository struct {
	db    *gorm.DB
	cache *cache.Redis
	log   *logger.Logger
}

func NewUserRepository(db *gorm.DB, cache *cache.Redis, log *logger.Logger) UserRepository {
	return &userRepository{
		db:    db,
		cache: cache,
		log:   log,
	}
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	err := r.db.WithContext(ctx).Create(user).Error
	if err != nil {
		r.log.Error("Failed to create user", zap.Error(err), zap.String("email", user.Email))
		return err
	}

	if r.cache != nil {
		_ = r.cache.Delete(ctx, constants.CacheKeyUsers)
	}

	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var user model.User

	if r.cache != nil {
		cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyUser, id.String())
		err := r.cache.Get(ctx, cacheKey, &user)
		if err == nil {
			return &user, nil
		}
	}

	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%s: %w", constants.ErrUserNotFound, err)
		}
		return nil, err
	}

	if r.cache != nil {
		cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyUser, id.String())
		_ = r.cache.Set(ctx, cacheKey, user, constants.CacheDefaultTTL)
	}

	return &user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User

	if r.cache != nil {
		cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyUser, email)
		err := r.cache.Get(ctx, cacheKey, &user)
		if err == nil {
			return &user, nil
		}
	}

	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%s: %w", constants.ErrUserNotFound, err)
		}
		return nil, err
	}

	if r.cache != nil {
		cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyUser, email)
		_ = r.cache.Set(ctx, cacheKey, user, constants.CacheDefaultTTL)
	}

	return &user, nil
}

func (r *userRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User

	if r.cache != nil {
		cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyUser, username)
		err := r.cache.Get(ctx, cacheKey, &user)
		if err == nil {
			return &user, nil
		}
	}

	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%s: %w", constants.ErrUserNotFound, err)
		}
		return nil, err
	}

	if r.cache != nil {
		cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyUser, username)
		_ = r.cache.Set(ctx, cacheKey, user, constants.CacheDefaultTTL)
	}

	return &user, nil
}

func (r *userRepository) GetByUsernameOrEmail(ctx context.Context, usernameOrEmail string) (*model.User, error) {
	var user model.User

	err := r.db.WithContext(ctx).Where("username = ? OR email = ?", usernameOrEmail, usernameOrEmail).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%s: %w", constants.ErrUserNotFound, err)
		}
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	err := r.db.WithContext(ctx).Save(user).Error
	if err != nil {
		r.log.Error("Failed to update user", zap.Error(err), zap.String("id", user.ID.String()))
		return err
	}

	if r.cache != nil {
		_ = r.cache.Delete(ctx, fmt.Sprintf("%s%s", constants.CacheKeyUser, user.ID.String()))
		_ = r.cache.Delete(ctx, fmt.Sprintf("%s%s", constants.CacheKeyUser, user.Email))
		_ = r.cache.Delete(ctx, fmt.Sprintf("%s%s", constants.CacheKeyUser, user.Username))
		_ = r.cache.Delete(ctx, constants.CacheKeyUsers)
	}

	return nil
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	user, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	err = r.db.WithContext(ctx).Delete(&model.User{}, id).Error
	if err != nil {
		r.log.Error("Failed to delete user", zap.Error(err), zap.String("id", id.String()))
		return err
	}

	if r.cache != nil {
		_ = r.cache.Delete(ctx, fmt.Sprintf("%s%s", constants.CacheKeyUser, id.String()))
		_ = r.cache.Delete(ctx, fmt.Sprintf("%s%s", constants.CacheKeyUser, user.Email))
		_ = r.cache.Delete(ctx, fmt.Sprintf("%s%s", constants.CacheKeyUser, user.Username))
		_ = r.cache.Delete(ctx, constants.CacheKeyUsers)
	}

	return nil
}

func (r *userRepository) List(ctx context.Context, filter *dto.UserFilter) ([]*model.User, int64, error) {
	var users []*model.User
	var count int64

	query := r.db.WithContext(ctx).Model(&model.User{})

	if filter.Role != "" {
		query = query.Where("role = ?", filter.Role)
	}

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if err := query.Count(&count).Error; err != nil {
		r.log.Error("Failed to count users", zap.Error(err))
		return nil, 0, err
	}

	if filter.SortBy != "" {
		if filter.Desc {
			query = query.Order(fmt.Sprintf("%s DESC", filter.SortBy))
		} else {
			query = query.Order(filter.SortBy)
		}
	} else {
		query = query.Order("created_at DESC")
	}

	query = query.Offset(filter.GetOffset()).Limit(filter.Limit)

	if err := query.Find(&users).Error; err != nil {
		r.log.Error("Failed to list users", zap.Error(err))
		return nil, 0, err
	}

	return users, count, nil
}

func (r *userRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	err := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).
		Update("last_login", time.Now()).Error
	if err != nil {
		r.log.Error("Failed to update last login", zap.Error(err), zap.String("id", id.String()))
		return err
	}

	if r.cache != nil {
		_ = r.cache.Delete(ctx, fmt.Sprintf("%s%s", constants.CacheKeyUser, id.String()))
	}

	return nil
}
