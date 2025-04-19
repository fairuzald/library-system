package service

import (
	"context"

	"github.com/fairuzald/library-system/services/book-service/internal/entity/dao"
	"github.com/fairuzald/library-system/services/book-service/internal/entity/dto"
	"github.com/google/uuid"
)

type BookService interface {
	CreateBook(ctx context.Context, req *dto.BookCreate) (*dao.BookResponse, error)
	GetBookByID(ctx context.Context, id uuid.UUID) (*dao.BookResponse, error)
	GetBookByISBN(ctx context.Context, isbn string) (*dao.BookResponse, error)
	UpdateBook(ctx context.Context, id uuid.UUID, req *dto.BookUpdate) (*dao.BookResponse, error)
	DeleteBook(ctx context.Context, id uuid.UUID) error
	ListBooks(ctx context.Context, filter *dto.BookFilter) (*dao.BookListResponse, error)
	SearchBooks(ctx context.Context, search *dto.BookSearch) (*dao.BookListResponse, error)
	GetBooksByCategory(ctx context.Context, categoryID string, page, limit int) (*dao.BookListResponse, error)
}
