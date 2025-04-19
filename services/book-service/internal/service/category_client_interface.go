package service

import (
	"context"
)

type CategoryClient interface {
	CategoryExists(ctx context.Context, categoryID string) (bool, error)

	GetCategoryName(ctx context.Context, categoryID string) (string, error)

	Close() error
}
