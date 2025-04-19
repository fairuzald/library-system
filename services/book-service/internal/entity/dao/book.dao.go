package dao

import (
	"time"

	"github.com/fairuzald/library-system/services/book-service/internal/entity/model"
	"github.com/google/uuid"
)

type BookResponse struct {
	ID                uuid.UUID `json:"id"`
	Title             string    `json:"title"`
	Author            string    `json:"author"`
	ISBN              string    `json:"isbn"`
	PublishedYear     int       `json:"published_year"`
	Publisher         string    `json:"publisher"`
	Description       string    `json:"description"`
	CategoryIDs       []string  `json:"category_ids,omitempty"`
	Language          string    `json:"language"`
	PageCount         int       `json:"page_count"`
	Status            string    `json:"status"`
	CoverImage        string    `json:"cover_image,omitempty"`
	AverageRating     float64   `json:"average_rating"`
	Quantity          int       `json:"quantity"`
	AvailableQuantity int       `json:"available_quantity"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func NewBookResponse(book *model.Book) *BookResponse {
	return &BookResponse{
		ID:                book.ID,
		Title:             book.Title,
		Author:            book.Author,
		ISBN:              book.ISBN,
		PublishedYear:     book.PublishedYear,
		Publisher:         book.Publisher,
		Description:       book.Description,
		CategoryIDs:       book.CategoryIDs,
		Language:          book.Language,
		PageCount:         book.PageCount,
		Status:            book.Status,
		CoverImage:        book.CoverImage,
		AverageRating:     book.AverageRating,
		Quantity:          book.Quantity,
		AvailableQuantity: book.AvailableQuantity,
		CreatedAt:         book.CreatedAt,
		UpdatedAt:         book.UpdatedAt,
	}
}

type BookListResponse struct {
	Books       []BookResponse `json:"books"`
	TotalItems  int64          `json:"total_items"`
	TotalPages  int            `json:"total_pages"`
	CurrentPage int            `json:"current_page"`
	PageSize    int            `json:"page_size"`
}
