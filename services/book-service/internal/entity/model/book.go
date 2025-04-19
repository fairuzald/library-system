package model

import (
	"time"

	"github.com/fairuzald/library-system/pkg/models"
	"github.com/google/uuid"
)

type Book struct {
	models.Base
	Title             string   `gorm:"type:varchar(255);not null" json:"title"`
	Author            string   `gorm:"type:varchar(255);not null" json:"author"`
	ISBN              string   `gorm:"type:varchar(20);uniqueIndex;not null" json:"isbn"`
	PublishedYear     int      `gorm:"not null" json:"published_year"`
	Publisher         string   `gorm:"type:varchar(255);not null" json:"publisher"`
	Description       string   `gorm:"type:text" json:"description"`
	Language          string   `gorm:"type:varchar(50);not null" json:"language"`
	PageCount         int      `gorm:"not null" json:"page_count"`
	Status            string   `gorm:"type:varchar(20);not null;default:'available'" json:"status"`
	CoverImage        string   `gorm:"type:text" json:"cover_image,omitempty"`
	AverageRating     float64  `gorm:"default:0" json:"average_rating"`
	Quantity          int      `gorm:"not null;default:1" json:"quantity"`
	AvailableQuantity int      `gorm:"not null;default:1" json:"available_quantity"`
	CategoryIDs       []string `gorm:"-" json:"category_ids,omitempty"`
}

func (Book) TableName() string {
	return "books"
}

type BookCategory struct {
	models.Base
	BookID     uuid.UUID `gorm:"type:uuid;not null" json:"book_id"`
	CategoryID uuid.UUID `gorm:"type:uuid;not null" json:"category_id"`
}

func (BookCategory) TableName() string {
	return "books_categories"
}

func NewBook(title, author, isbn string, publishedYear int, publisher, description, language string, pageCount int, categoryIDs []string) *Book {
	return &Book{
		Base: models.Base{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Title:             title,
		Author:            author,
		ISBN:              isbn,
		PublishedYear:     publishedYear,
		Publisher:         publisher,
		Description:       description,
		Language:          language,
		PageCount:         pageCount,
		Status:            "available",
		AverageRating:     0,
		Quantity:          1,
		AvailableQuantity: 1,
		CategoryIDs:       categoryIDs,
	}
}
