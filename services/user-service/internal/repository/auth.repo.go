package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/fairuzald/library-system/pkg/cache"
	"github.com/fairuzald/library-system/pkg/logger"
	"github.com/fairuzald/library-system/services/user-service/internal/entity/model"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AuthRepository interface {
	StoreRefreshToken(ctx context.Context, userID uuid.UUID, token string, expiration time.Time) error
	GetUserByRefreshToken(ctx context.Context, token string) (*model.User, error)
	RevokeRefreshToken(ctx context.Context, userID uuid.UUID) error
	StoreTokenInBlacklist(ctx context.Context, token string, expiration time.Duration) error
	IsTokenBlacklisted(ctx context.Context, token string) (bool, error)
	CleanupExpiredTokens(ctx context.Context) error
}

type authRepository struct {
	db    *gorm.DB
	cache *cache.Redis
	log   *logger.Logger
}

func NewAuthRepository(db *gorm.DB, cache *cache.Redis, log *logger.Logger) AuthRepository {
	return &authRepository{
		db:    db,
		cache: cache,
		log:   log,
	}
}

func (r *authRepository) StoreRefreshToken(ctx context.Context, userID uuid.UUID, token string, expiration time.Time) error {
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"refresh_token":     token,
			"refresh_token_exp": expiration,
		}).Error
}

func (r *authRepository) GetUserByRefreshToken(ctx context.Context, token string) (*model.User, error) {
	var user model.User

	err := r.db.WithContext(ctx).
		Where("refresh_token = ? AND refresh_token_exp > ?", token, time.Now()).
		First(&user).Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *authRepository) RevokeRefreshToken(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"refresh_token":     "",
			"refresh_token_exp": time.Now(),
		}).Error
}

func (r *authRepository) StoreTokenInBlacklist(ctx context.Context, token string, expiration time.Duration) error {
	if r.cache == nil {
		r.log.Warn("Cache not available for token blacklist")
		return nil
	}

	key := fmt.Sprintf("blacklist:%s", token)
	return r.cache.Set(ctx, key, true, expiration)
}

func (r *authRepository) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	if r.cache == nil {
		return false, nil
	}

	key := fmt.Sprintf("blacklist:%s", token)
	var blacklisted bool

	err := r.cache.Get(ctx, key, &blacklisted)
	if err != nil {
		return false, nil
	}

	return blacklisted, nil
}

func (r *authRepository) CleanupExpiredTokens(ctx context.Context) error {
	err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("refresh_token_exp < ?", time.Now()).
		Updates(map[string]interface{}{
			"refresh_token":     "",
			"refresh_token_exp": time.Now(),
		}).Error

	if err != nil {
		r.log.Error("Failed to cleanup expired tokens", zap.Error(err))
		return err
	}

	return nil
}
