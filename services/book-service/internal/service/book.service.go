package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/fairuzald/library-system/pkg/constants"
	"github.com/fairuzald/library-system/pkg/logger"
	"github.com/fairuzald/library-system/services/book-service/internal/entity/dao"
	"github.com/fairuzald/library-system/services/book-service/internal/entity/dto"
	"github.com/fairuzald/library-system/services/book-service/internal/entity/model"
	"github.com/fairuzald/library-system/services/book-service/internal/repository"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type bookService struct {
	bookRepo     repository.BookRepository
	categoryGRPC CategoryClient
	log          *logger.Logger
}

func NewBookService(bookRepo repository.BookRepository, categoryGRPC CategoryClient, log *logger.Logger) BookService {
	return &bookService{
		bookRepo:     bookRepo,
		categoryGRPC: categoryGRPC,
		log:          log,
	}
}

func (s *bookService) CreateBook(ctx context.Context, req *dto.BookCreate) (*dao.BookResponse, error) {
	if req.ISBN != "" {
		existingBook, err := s.bookRepo.GetByISBN(ctx, req.ISBN)
		if err == nil && existingBook != nil {
			return nil, fmt.Errorf("book with ISBN %s already exists", req.ISBN)
		}
	}

	// Validate category IDs if provided
	if len(req.CategoryIDs) > 0 && s.categoryGRPC != nil {
		for _, catID := range req.CategoryIDs {
			if catID == "" {
				continue
			}

			_, err := uuid.Parse(catID)
			if err != nil {
				return nil, fmt.Errorf("invalid category ID format: %s", catID)
			}

			exists, err := s.categoryGRPC.CategoryExists(ctx, catID)
			if err != nil {
				s.log.Warn("Failed to validate category ID", zap.Error(err), zap.String("category_id", catID))
			} else if !exists {
				return nil, fmt.Errorf("category with ID %s does not exist", catID)
			}
		}
	}

	book := model.NewBook(
		req.Title,
		req.Author,
		req.ISBN,
		req.PublishedYear,
		req.Publisher,
		req.Description,
		req.Language,
		req.PageCount,
		req.CategoryIDs,
	)

	if req.CoverImage != "" {
		book.CoverImage = req.CoverImage
	}

	if req.Quantity > 0 {
		book.Quantity = req.Quantity
		book.AvailableQuantity = req.Quantity
	}

	if err := s.bookRepo.Create(ctx, book); err != nil {
		s.log.Error("Failed to create book", zap.Error(err))
		return nil, errors.New(constants.ErrInternalServer)
	}

	return dao.NewBookResponse(book), nil
}

func (s *bookService) DeleteBook(ctx context.Context, id uuid.UUID) error {
	_, err := s.bookRepo.GetByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), constants.ErrBookNotFound) {
			return errors.New(constants.ErrBookNotFound)
		}
		s.log.Error("Failed to get book for deletion", zap.Error(err), zap.String("id", id.String()))
		return errors.New(constants.ErrInternalServer)
	}

	if err := s.bookRepo.Delete(ctx, id); err != nil {
		s.log.Error("Failed to delete book", zap.Error(err), zap.String("id", id.String()))
		return errors.New(constants.ErrInternalServer)
	}

	return nil
}

func (s *bookService) ListBooks(ctx context.Context, filter *dto.BookFilter) (*dao.BookListResponse, error) {
	filter.Validate()

	books, count, err := s.bookRepo.List(ctx, filter)
	if err != nil {
		s.log.Error("Failed to list books", zap.Error(err))
		return nil, errors.New(constants.ErrInternalServer)
	}

	response := &dao.BookListResponse{
		Books:       make([]dao.BookResponse, 0, len(books)),
		TotalItems:  count,
		TotalPages:  (int(count) + filter.Limit - 1) / filter.Limit,
		CurrentPage: filter.Page,
		PageSize:    filter.Limit,
	}

	for _, book := range books {
		response.Books = append(response.Books, *dao.NewBookResponse(book))
	}

	return response, nil
}

func (s *bookService) SearchBooks(ctx context.Context, search *dto.BookSearch) (*dao.BookListResponse, error) {
	if search.Page <= 0 {
		search.Page = 1
	}

	if search.Limit <= 0 {
		search.Limit = constants.DefaultPageSize
	} else if search.Limit > constants.MaxPageSize {
		search.Limit = constants.MaxPageSize
	}

	books, count, err := s.bookRepo.Search(ctx, search)
	if err != nil {
		s.log.Error("Failed to search books", zap.Error(err))
		return nil, errors.New(constants.ErrInternalServer)
	}

	response := &dao.BookListResponse{
		Books:       make([]dao.BookResponse, 0, len(books)),
		TotalItems:  count,
		TotalPages:  (int(count) + search.Limit - 1) / search.Limit,
		CurrentPage: search.Page,
		PageSize:    search.Limit,
	}

	for _, book := range books {
		response.Books = append(response.Books, *dao.NewBookResponse(book))
	}

	return response, nil
}

func (s *bookService) GetBooksByCategory(ctx context.Context, categoryID string, page, limit int) (*dao.BookListResponse, error) {
	if page <= 0 {
		page = 1
	}

	if limit <= 0 {
		limit = constants.DefaultPageSize
	} else if limit > constants.MaxPageSize {
		limit = constants.MaxPageSize
	}

	// Validate category ID format
	_, err := uuid.Parse(categoryID)
	if err != nil {
		return nil, fmt.Errorf("invalid category ID format: %s", categoryID)
	}

	// Check if category exists
	if s.categoryGRPC != nil {
		exists, err := s.categoryGRPC.CategoryExists(ctx, categoryID)
		if err != nil {
			s.log.Warn("Failed to validate category ID", zap.Error(err), zap.String("category_id", categoryID))
		} else if !exists {
			return nil, fmt.Errorf("category with ID %s does not exist", categoryID)
		}
	}

	books, count, err := s.bookRepo.GetByCategory(ctx, categoryID, page, limit)
	if err != nil {
		s.log.Error("Failed to get books by category", zap.Error(err), zap.String("category_id", categoryID))
		return nil, errors.New(constants.ErrInternalServer)
	}

	response := &dao.BookListResponse{
		Books:       make([]dao.BookResponse, 0, len(books)),
		TotalItems:  count,
		TotalPages:  (int(count) + limit - 1) / limit,
		CurrentPage: page,
		PageSize:    limit,
	}

	for _, book := range books {
		response.Books = append(response.Books, *dao.NewBookResponse(book))
	}

	return response, nil
}

func (s *bookService) GetBookByID(ctx context.Context, id uuid.UUID) (*dao.BookResponse, error) {
	book, err := s.bookRepo.GetByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), constants.ErrBookNotFound) {
			return nil, errors.New(constants.ErrBookNotFound)
		}
		s.log.Error("Failed to get book", zap.Error(err), zap.String("id", id.String()))
		return nil, errors.New(constants.ErrInternalServer)
	}

	return dao.NewBookResponse(book), nil
}

func (s *bookService) GetBookByISBN(ctx context.Context, isbn string) (*dao.BookResponse, error) {
	book, err := s.bookRepo.GetByISBN(ctx, isbn)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, fmt.Errorf("book with ISBN %s not found", isbn)
		}
		s.log.Error("Failed to get book by ISBN", zap.Error(err), zap.String("isbn", isbn))
		return nil, errors.New(constants.ErrInternalServer)
	}

	return dao.NewBookResponse(book), nil
}

func (s *bookService) UpdateBook(ctx context.Context, id uuid.UUID, req *dto.BookUpdate) (*dao.BookResponse, error) {
	book, err := s.bookRepo.GetByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), constants.ErrBookNotFound) {
			return nil, errors.New(constants.ErrBookNotFound)
		}
		s.log.Error("Failed to get book for update", zap.Error(err), zap.String("id", id.String()))
		return nil, errors.New(constants.ErrInternalServer)
	}

	// Check ISBN uniqueness if updating
	if req.ISBN != nil && *req.ISBN != book.ISBN {
		existingBook, err := s.bookRepo.GetByISBN(ctx, *req.ISBN)
		if err == nil && existingBook != nil && existingBook.ID != id {
			return nil, fmt.Errorf("book with ISBN %s already exists", *req.ISBN)
		}
	}

	// Update book fields
	if req.Title != nil {
		book.Title = *req.Title
	}

	if req.Author != nil {
		book.Author = *req.Author
	}

	if req.ISBN != nil {
		book.ISBN = *req.ISBN
	}

	if req.PublishedYear != nil {
		book.PublishedYear = *req.PublishedYear
	}

	if req.Publisher != nil {
		book.Publisher = *req.Publisher
	}

	if req.Description != nil {
		book.Description = *req.Description
	}

	if req.Language != nil {
		book.Language = *req.Language
	}

	if req.PageCount != nil {
		book.PageCount = *req.PageCount
	}

	if req.Status != nil {
		book.Status = *req.Status
	}

	if req.CoverImage != nil {
		book.CoverImage = *req.CoverImage
	}

	if req.Quantity != nil {
		book.Quantity = *req.Quantity
		// Only update available quantity if it hasn't been explicitly set
		if req.AvailableQuantity == nil {
			difference := *req.Quantity - book.Quantity
			book.AvailableQuantity += difference
			if book.AvailableQuantity < 0 {
				book.AvailableQuantity = 0
			}
		}
	}

	if req.AvailableQuantity != nil {
		if *req.AvailableQuantity > book.Quantity {
			return nil, fmt.Errorf("available quantity cannot exceed total quantity")
		}
		book.AvailableQuantity = *req.AvailableQuantity
	}

	// Validate and update category IDs if provided
	if len(req.CategoryIDs) > 0 {
		if s.categoryGRPC != nil {
			for _, catID := range req.CategoryIDs {
				if catID == "" {
					continue
				}

				_, err := uuid.Parse(catID)
				if err != nil {
					return nil, fmt.Errorf("invalid category ID format: %s", catID)
				}

				exists, err := s.categoryGRPC.CategoryExists(ctx, catID)
				if err != nil {
					s.log.Warn("Failed to validate category ID", zap.Error(err), zap.String("category_id", catID))
				} else if !exists {
					return nil, fmt.Errorf("category with ID %s does not exist", catID)
				}
			}
		}

		book.CategoryIDs = req.CategoryIDs
	}

	if err := s.bookRepo.Update(ctx, book); err != nil {
		s.log.Error("Failed to update book", zap.Error(err), zap.String("id", id.String()))
		return nil, errors.New(constants.ErrInternalServer)
	}

	return dao.NewBookResponse(book), nil
}
