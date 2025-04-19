package dto

import (
	"github.com/fairuzald/library-system/pkg/constants"
)

type BookCreate struct {
	Title         string   `json:"title" validate:"required"`
	Author        string   `json:"author" validate:"required"`
	ISBN          string   `json:"isbn" validate:"required"`
	PublishedYear int      `json:"published_year" validate:"required,gt=0"`
	Publisher     string   `json:"publisher" validate:"required"`
	Description   string   `json:"description"`
	CategoryIDs   []string `json:"category_ids"`
	Language      string   `json:"language" validate:"required"`
	PageCount     int      `json:"page_count" validate:"required,gt=0"`
	CoverImage    string   `json:"cover_image,omitempty"`
	Quantity      int      `json:"quantity,omitempty" validate:"omitempty,gt=0"`
}

type BookUpdate struct {
	Title             *string  `json:"title,omitempty"`
	Author            *string  `json:"author,omitempty"`
	ISBN              *string  `json:"isbn,omitempty"`
	PublishedYear     *int     `json:"published_year,omitempty" validate:"omitempty,gt=0"`
	Publisher         *string  `json:"publisher,omitempty"`
	Description       *string  `json:"description,omitempty"`
	CategoryIDs       []string `json:"category_ids,omitempty"`
	Language          *string  `json:"language,omitempty"`
	PageCount         *int     `json:"page_count,omitempty" validate:"omitempty,gt=0"`
	Status            *string  `json:"status,omitempty" validate:"omitempty,oneof=available borrowed reserved maintenance"`
	CoverImage        *string  `json:"cover_image,omitempty"`
	Quantity          *int     `json:"quantity,omitempty" validate:"omitempty,gt=0"`
	AvailableQuantity *int     `json:"available_quantity,omitempty" validate:"omitempty,gte=0"`
}

type BookFilter struct {
	Page       int    `form:"page,default=1" query:"page,default=1"`
	Limit      int    `form:"limit,default=10" query:"limit,default=10"`
	SortBy     string `form:"sort_by,default=created_at" query:"sort_by,default=created_at"`
	Desc       bool   `form:"desc" query:"desc"`
	Status     string `form:"status" query:"status"`
	CategoryID string `form:"category_id" query:"category_id"`
	Author     string `form:"author" query:"author"`
	Language   string `form:"language" query:"language"`
}

type BookSearch struct {
	Query string `form:"query" validate:"required"`
	Field string `form:"field"`
	Page  int    `form:"page,default=1" query:"page,default=1"`
	Limit int    `form:"limit,default=10" query:"limit,default=10"`
}

func (f *BookFilter) Validate() {
	if f.Page <= 0 {
		f.Page = 1
	}

	if f.Limit <= 0 {
		f.Limit = constants.DefaultPageSize
	} else if f.Limit > constants.MaxPageSize {
		f.Limit = constants.MaxPageSize
	}

	if f.Status != "" && f.Status != constants.BookStatusAvailable && f.Status != constants.BookStatusBorrowed &&
		f.Status != constants.BookStatusReserved && f.Status != constants.BookStatusMaintenance {
		f.Status = ""
	}
}

func (f *BookFilter) GetOffset() int {
	return (f.Page - 1) * f.Limit
}
