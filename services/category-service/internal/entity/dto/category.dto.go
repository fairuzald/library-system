package dto

import (
	"github.com/fairuzald/library-system/pkg/constants"
)

type CategoryCreate struct {
	Name        string  `json:"name" validate:"required"`
	Description string  `json:"description"`
	ParentID    *string `json:"parent_id,omitempty"`
}

type CategoryUpdate struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	ParentID    *string `json:"parent_id,omitempty"`
}

type CategoryFilter struct {
	Page     int     `form:"page,default=1" query:"page,default=1"`
	Limit    int     `form:"limit,default=10" query:"limit,default=10"`
	SortBy   string  `form:"sort_by,default=name" query:"sort_by,default=name"`
	Desc     bool    `form:"desc" query:"desc"`
	ParentID *string `form:"parent_id" query:"parent_id"`
}

func (f *CategoryFilter) Validate() {
	if f.Page <= 0 {
		f.Page = 1
	}

	if f.Limit <= 0 {
		f.Limit = constants.DefaultPageSize
	} else if f.Limit > constants.MaxPageSize {
		f.Limit = constants.MaxPageSize
	}

	if f.SortBy == "" {
		f.SortBy = "name"
	}
}

func (f *CategoryFilter) GetOffset() int {
	return (f.Page - 1) * f.Limit
}
