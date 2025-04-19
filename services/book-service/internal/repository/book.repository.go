package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/fairuzald/library-system/pkg/cache"
	"github.com/fairuzald/library-system/pkg/constants"
	"github.com/fairuzald/library-system/pkg/logger"
	"github.com/fairuzald/library-system/services/book-service/internal/entity/dto"
	"github.com/fairuzald/library-system/services/book-service/internal/entity/model"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BookRepository interface {
	Create(ctx context.Context, book *model.Book) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Book, error)
	GetByISBN(ctx context.Context, isbn string) (*model.Book, error)
	Update(ctx context.Context, book *model.Book) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter *dto.BookFilter) ([]*model.Book, int64, error)
	Search(ctx context.Context, search *dto.BookSearch) ([]*model.Book, int64, error)
	GetByCategory(ctx context.Context, categoryID string, page, limit int) ([]*model.Book, int64, error)
	AddCategories(ctx context.Context, bookID uuid.UUID, categoryIDs []string) error
	RemoveCategories(ctx context.Context, bookID uuid.UUID) error
	GetBookCategories(ctx context.Context, bookID uuid.UUID) ([]string, error)
}

type bookRepository struct {
	db    *gorm.DB
	cache *cache.Redis
	log   *logger.Logger
}

func NewBookRepository(db *gorm.DB, cache *cache.Redis, log *logger.Logger) BookRepository {
	return &bookRepository{
		db:    db,
		cache: cache,
		log:   log,
	}
}

func (r *bookRepository) Create(ctx context.Context, book *model.Book) error {
	tx := r.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Create(book).Error; err != nil {
		tx.Rollback()
		r.log.Error("Failed to create book", zap.Error(err), zap.String("title", book.Title))
		return err
	}

	if len(book.CategoryIDs) > 0 {
		if err := r.addBookCategories(tx, book.ID, book.CategoryIDs); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		r.log.Error("Failed to commit transaction", zap.Error(err))
		return err
	}

	if r.cache != nil {
		cacheKey := fmt.Sprintf("%slist", constants.CacheKeyBooks)
		_ = r.cache.Delete(ctx, cacheKey)
	}

	return nil
}

func (r *bookRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Book, error) {
	var book model.Book

	if r.cache != nil {
		cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyBook, id.String())
		err := r.cache.Get(ctx, cacheKey, &book)
		if err == nil {
			categoryIDs, _ := r.GetBookCategories(ctx, id)
			book.CategoryIDs = categoryIDs
			return &book, nil
		}
	}

	err := r.db.WithContext(ctx).Where("id = ?", id).First(&book).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%s: %w", constants.ErrBookNotFound, err)
		}
		return nil, err
	}

	categoryIDs, err := r.GetBookCategories(ctx, id)
	if err != nil {
		r.log.Error("Failed to get book categories", zap.Error(err), zap.String("book_id", id.String()))
	}
	book.CategoryIDs = categoryIDs

	if r.cache != nil {
		cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyBook, id.String())
		_ = r.cache.Set(ctx, cacheKey, book, constants.CacheDefaultTTL)
	}

	return &book, nil
}

func (r *bookRepository) GetByISBN(ctx context.Context, isbn string) (*model.Book, error) {
	var book model.Book

	if r.cache != nil {
		cacheKey := fmt.Sprintf("%sisbn:%s", constants.CacheKeyBook, isbn)
		err := r.cache.Get(ctx, cacheKey, &book)
		if err == nil {
			categoryIDs, _ := r.GetBookCategories(ctx, book.ID)
			book.CategoryIDs = categoryIDs
			return &book, nil
		}
	}

	err := r.db.WithContext(ctx).Where("isbn = ?", isbn).First(&book).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("book with ISBN %s not found: %w", isbn, err)
		}
		return nil, err
	}

	categoryIDs, err := r.GetBookCategories(ctx, book.ID)
	if err != nil {
		r.log.Error("Failed to get book categories", zap.Error(err), zap.String("isbn", isbn))
	}
	book.CategoryIDs = categoryIDs

	if r.cache != nil {
		cacheKey := fmt.Sprintf("%sisbn:%s", constants.CacheKeyBook, isbn)
		_ = r.cache.Set(ctx, cacheKey, book, constants.CacheDefaultTTL)
	}

	return &book, nil
}

func (r *bookRepository) Update(ctx context.Context, book *model.Book) error {
	tx := r.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Save(book).Error; err != nil {
		tx.Rollback()
		r.log.Error("Failed to update book", zap.Error(err), zap.String("id", book.ID.String()))
		return err
	}

	if err := r.RemoveCategories(ctx, book.ID); err != nil {
		tx.Rollback()
		return err
	}

	if len(book.CategoryIDs) > 0 {
		if err := r.addBookCategories(tx, book.ID, book.CategoryIDs); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		r.log.Error("Failed to commit transaction", zap.Error(err))
		return err
	}

	if r.cache != nil {
		cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyBook, book.ID.String())
		_ = r.cache.Delete(ctx, cacheKey)
		cacheKey = fmt.Sprintf("%sisbn:%s", constants.CacheKeyBook, book.ISBN)
		_ = r.cache.Delete(ctx, cacheKey)
		cacheKey = fmt.Sprintf("%slist", constants.CacheKeyBooks)
		_ = r.cache.Delete(ctx, cacheKey)
	}

	return nil
}

func (r *bookRepository) Delete(ctx context.Context, id uuid.UUID) error {
	book, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	tx := r.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := r.RemoveCategories(ctx, id); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Delete(&model.Book{}, id).Error; err != nil {
		tx.Rollback()
		r.log.Error("Failed to delete book", zap.Error(err), zap.String("id", id.String()))
		return err
	}

	if err := tx.Commit().Error; err != nil {
		r.log.Error("Failed to commit transaction", zap.Error(err))
		return err
	}

	if r.cache != nil {
		cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyBook, id.String())
		_ = r.cache.Delete(ctx, cacheKey)
		cacheKey = fmt.Sprintf("%sisbn:%s", constants.CacheKeyBook, book.ISBN)
		_ = r.cache.Delete(ctx, cacheKey)
		cacheKey = fmt.Sprintf("%slist", constants.CacheKeyBooks)
		_ = r.cache.Delete(ctx, cacheKey)
	}

	return nil
}

func (r *bookRepository) List(ctx context.Context, filter *dto.BookFilter) ([]*model.Book, int64, error) {
	var books []*model.Book
	var count int64

	query := r.db.WithContext(ctx).Model(&model.Book{})

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if filter.Author != "" {
		query = query.Where("author ILIKE ?", "%"+filter.Author+"%")
	}

	if filter.Language != "" {
		query = query.Where("language = ?", filter.Language)
	}

	if filter.CategoryID != "" {
		subQuery := r.db.Table("books_categories").
			Select("book_id").
			Where("category_id = ?", filter.CategoryID)
		query = query.Where("id IN (?)", subQuery)
	}

	if err := query.Count(&count).Error; err != nil {
		r.log.Error("Failed to count books", zap.Error(err))
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

	if err := query.Find(&books).Error; err != nil {
		r.log.Error("Failed to list books", zap.Error(err))
		return nil, 0, err
	}

	for _, book := range books {
		categoryIDs, err := r.GetBookCategories(ctx, book.ID)
		if err != nil {
			r.log.Error("Failed to get book categories", zap.Error(err), zap.String("book_id", book.ID.String()))
			continue
		}
		book.CategoryIDs = categoryIDs
	}

	return books, count, nil
}

func (r *bookRepository) Search(ctx context.Context, search *dto.BookSearch) ([]*model.Book, int64, error) {
	var books []*model.Book
	var count int64

	query := r.db.WithContext(ctx).Model(&model.Book{})

	searchTerm := "%" + search.Query + "%"
	if search.Field != "" {
		switch search.Field {
		case "title":
			query = query.Where("title ILIKE ?", searchTerm)
		case "author":
			query = query.Where("author ILIKE ?", searchTerm)
		case "isbn":
			query = query.Where("isbn LIKE ?", searchTerm)
		case "publisher":
			query = query.Where("publisher ILIKE ?", searchTerm)
		case "description":
			query = query.Where("description ILIKE ?", searchTerm)
		default:
			query = query.Where("title ILIKE ? OR author ILIKE ? OR isbn LIKE ? OR publisher ILIKE ? OR description ILIKE ?",
				searchTerm, searchTerm, searchTerm, searchTerm, searchTerm)
		}
	} else {
		query = query.Where("title ILIKE ? OR author ILIKE ? OR isbn LIKE ? OR publisher ILIKE ? OR description ILIKE ?",
			searchTerm, searchTerm, searchTerm, searchTerm, searchTerm)
	}

	if err := query.Count(&count).Error; err != nil {
		r.log.Error("Failed to count search results", zap.Error(err))
		return nil, 0, err
	}

	offset := (search.Page - 1) * search.Limit
	if err := query.Offset(offset).Limit(search.Limit).Find(&books).Error; err != nil {
		r.log.Error("Failed to search books", zap.Error(err))
		return nil, 0, err
	}

	for _, book := range books {
		categoryIDs, err := r.GetBookCategories(ctx, book.ID)
		if err != nil {
			r.log.Error("Failed to get book categories", zap.Error(err), zap.String("book_id", book.ID.String()))
			continue
		}
		book.CategoryIDs = categoryIDs
	}

	return books, count, nil
}

func (r *bookRepository) GetByCategory(ctx context.Context, categoryID string, page, limit int) ([]*model.Book, int64, error) {
	var books []*model.Book
	var count int64

	catUUID, err := uuid.Parse(categoryID)
	if err != nil {
		return nil, 0, err
	}

	subQuery := r.db.Table("books_categories").
		Select("book_id").
		Where("category_id = ?", catUUID)

	query := r.db.WithContext(ctx).Model(&model.Book{}).Where("id IN (?)", subQuery)

	if err := query.Count(&count).Error; err != nil {
		r.log.Error("Failed to count books by category", zap.Error(err))
		return nil, 0, err
	}

	offset := (page - 1) * limit
	if err := query.Offset(offset).Limit(limit).Find(&books).Error; err != nil {
		r.log.Error("Failed to get books by category", zap.Error(err))
		return nil, 0, err
	}

	for _, book := range books {
		categoryIDs, err := r.GetBookCategories(ctx, book.ID)
		if err != nil {
			r.log.Error("Failed to get book categories", zap.Error(err), zap.String("book_id", book.ID.String()))
			continue
		}
		book.CategoryIDs = categoryIDs
	}

	return books, count, nil
}

func (r *bookRepository) AddCategories(ctx context.Context, bookID uuid.UUID, categoryIDs []string) error {
	tx := r.db.WithContext(ctx)
	return r.addBookCategories(tx, bookID, categoryIDs)
}

func (r *bookRepository) RemoveCategories(ctx context.Context, bookID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("book_id = ?", bookID).Delete(&model.BookCategory{}).Error
}

func (r *bookRepository) GetBookCategories(ctx context.Context, bookID uuid.UUID) ([]string, error) {
	var bookCategories []model.BookCategory
	var categoryIDs []string

	err := r.db.WithContext(ctx).Where("book_id = ?", bookID).Find(&bookCategories).Error
	if err != nil {
		return nil, err
	}

	for _, bc := range bookCategories {
		categoryIDs = append(categoryIDs, bc.CategoryID.String())
	}

	return categoryIDs, nil
}

func (r *bookRepository) addBookCategories(tx *gorm.DB, bookID uuid.UUID, categoryIDs []string) error {
	for _, catID := range categoryIDs {
		if catID == "" {
			continue
		}

		catUUID, err := uuid.Parse(strings.TrimSpace(catID))
		if err != nil {
			r.log.Warn("Invalid category ID format", zap.String("category_id", catID))
			continue
		}

		bookCategory := model.BookCategory{
			BookID:     bookID,
			CategoryID: catUUID,
		}

		if err := tx.Create(&bookCategory).Error; err != nil {
			if !strings.Contains(err.Error(), "duplicate key") {
				r.log.Error("Failed to add book category", zap.Error(err))
				return err
			}
		}
	}
	return nil
}
