package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/fairuzald/library-system/pkg/constants"
	"github.com/fairuzald/library-system/pkg/logger"
	"github.com/fairuzald/library-system/pkg/middleware"
	"github.com/fairuzald/library-system/pkg/utils"
	"github.com/fairuzald/library-system/services/book-service/internal/entity/dto"
	"github.com/fairuzald/library-system/services/book-service/internal/service"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type BookHandler struct {
	bookService service.BookService
	log         *logger.Logger
}

func NewBookHandler(bookService service.BookService, log *logger.Logger) *BookHandler {
	return &BookHandler{
		bookService: bookService,
		log:         log,
	}
}

func (h *BookHandler) isAdminOrLibrarian(r *http.Request) bool {
	role, ok := r.Context().Value(middleware.UserRoleKey).(string)
	if !ok {
		return false
	}
	return role == constants.RoleAdmin || role == constants.RoleLibrarian
}

func (h *BookHandler) HandleCreateBook(w http.ResponseWriter, r *http.Request) {
	if !h.isAdminOrLibrarian(r) {
		utils.RespondWithError(w, http.StatusForbidden, constants.ErrForbidden, nil)
		return
	}

	var req dto.BookCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("Failed to decode request body", zap.Error(err))
		utils.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidRequest, err)
		return
	}

	if validationErrors, err := utils.Validate(req); err != nil {
		h.log.Info("Validation failed for create book request", zap.Any("errors", validationErrors))
		utils.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidField, err)
		return
	}

	book, err := h.bookService.CreateBook(r.Context(), &req)
	if err != nil {
		h.log.Error("Failed to create book", zap.Error(err))

		if err.Error() == constants.ErrInternalServer {
			utils.RespondWithError(w, http.StatusInternalServerError, constants.ErrInternalServer, nil)
		} else {
			utils.RespondWithError(w, http.StatusBadRequest, err.Error(), nil)
		}
		return
	}

	utils.RespondWithSuccess(w, http.StatusCreated, "Book created successfully", book)
}

func (h *BookHandler) HandleGetBook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid book ID", err)
		return
	}

	book, err := h.bookService.GetBookByID(r.Context(), id)
	if err != nil {
		if err.Error() == constants.ErrBookNotFound {
			utils.RespondWithError(w, http.StatusNotFound, constants.ErrBookNotFound, nil)
			return
		}

		h.log.Error("Failed to get book", zap.Error(err), zap.String("id", id.String()))
		utils.RespondWithError(w, http.StatusInternalServerError, constants.ErrInternalServer, nil)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Book retrieved successfully", book)
}

func (h *BookHandler) HandleGetBookByISBN(w http.ResponseWriter, r *http.Request) {
	isbn := r.URL.Query().Get("isbn")
	if isbn == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "ISBN parameter is required", nil)
		return
	}

	book, err := h.bookService.GetBookByISBN(r.Context(), isbn)
	if err != nil {
		if err.Error() == "book with ISBN "+isbn+" not found" {
			utils.RespondWithError(w, http.StatusNotFound, err.Error(), nil)
			return
		}

		h.log.Error("Failed to get book by ISBN", zap.Error(err), zap.String("isbn", isbn))
		utils.RespondWithError(w, http.StatusInternalServerError, constants.ErrInternalServer, nil)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Book retrieved successfully", book)
}

func (h *BookHandler) HandleUpdateBook(w http.ResponseWriter, r *http.Request) {
	if !h.isAdminOrLibrarian(r) {
		utils.RespondWithError(w, http.StatusForbidden, constants.ErrForbidden, nil)
		return
	}

	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid book ID", err)
		return
	}

	var req dto.BookUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("Failed to decode request body", zap.Error(err))
		utils.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidRequest, err)
		return
	}

	if validationErrors, err := utils.Validate(req); err != nil {
		h.log.Info("Validation failed for update book request", zap.Any("errors", validationErrors))
		utils.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidField, err)
		return
	}

	book, err := h.bookService.UpdateBook(r.Context(), id, &req)
	if err != nil {
		if err.Error() == constants.ErrBookNotFound {
			utils.RespondWithError(w, http.StatusNotFound, constants.ErrBookNotFound, nil)
			return
		}

		h.log.Error("Failed to update book", zap.Error(err), zap.String("id", id.String()))

		if err.Error() == constants.ErrInternalServer {
			utils.RespondWithError(w, http.StatusInternalServerError, constants.ErrInternalServer, nil)
		} else {
			utils.RespondWithError(w, http.StatusBadRequest, err.Error(), nil)
		}
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Book updated successfully", book)
}

func (h *BookHandler) HandleDeleteBook(w http.ResponseWriter, r *http.Request) {
	if !h.isAdminOrLibrarian(r) {
		utils.RespondWithError(w, http.StatusForbidden, constants.ErrForbidden, nil)
		return
	}

	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid book ID", err)
		return
	}

	if err := h.bookService.DeleteBook(r.Context(), id); err != nil {
		if err.Error() == constants.ErrBookNotFound {
			utils.RespondWithError(w, http.StatusNotFound, constants.ErrBookNotFound, nil)
			return
		}

		h.log.Error("Failed to delete book", zap.Error(err), zap.String("id", id.String()))
		utils.RespondWithError(w, http.StatusInternalServerError, constants.ErrInternalServer, nil)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Book deleted successfully", nil)
}

func (h *BookHandler) HandleListBooks(w http.ResponseWriter, r *http.Request) {
	filter := &dto.BookFilter{
		Status:     r.URL.Query().Get("status"),
		CategoryID: r.URL.Query().Get("category_id"),
		Author:     r.URL.Query().Get("author"),
		Language:   r.URL.Query().Get("language"),
		SortBy:     r.URL.Query().Get("sort_by"),
	}

	if page := r.URL.Query().Get("page"); page != "" {
		if pageNum, err := strconv.Atoi(page); err == nil {
			filter.Page = pageNum
		}
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if limitNum, err := strconv.Atoi(limit); err == nil {
			filter.Limit = limitNum
		}
	}

	if desc := r.URL.Query().Get("desc"); desc == "true" {
		filter.Desc = true
	}

	books, err := h.bookService.ListBooks(r.Context(), filter)
	if err != nil {
		h.log.Error("Failed to list books", zap.Error(err))
		utils.RespondWithError(w, http.StatusInternalServerError, constants.ErrInternalServer, nil)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Books retrieved successfully", books)
}

func (h *BookHandler) HandleSearchBooks(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Query parameter is required", nil)
		return
	}

	search := &dto.BookSearch{
		Query: query,
		Field: r.URL.Query().Get("field"),
	}

	if page := r.URL.Query().Get("page"); page != "" {
		if pageNum, err := strconv.Atoi(page); err == nil {
			search.Page = pageNum
		}
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if limitNum, err := strconv.Atoi(limit); err == nil {
			search.Limit = limitNum
		}
	}

	books, err := h.bookService.SearchBooks(r.Context(), search)
	if err != nil {
		h.log.Error("Failed to search books", zap.Error(err))
		utils.RespondWithError(w, http.StatusInternalServerError, constants.ErrInternalServer, nil)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Search results retrieved successfully", books)
}

func (h *BookHandler) HandleGetBooksByCategory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	categoryID := vars["categoryId"]

	if categoryID == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Category ID parameter is required", nil)
		return
	}

	page := 1
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if pageNum, err := strconv.Atoi(pageStr); err == nil && pageNum > 0 {
			page = pageNum
		}
	}

	limit := constants.DefaultPageSize
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limitNum, err := strconv.Atoi(limitStr); err == nil && limitNum > 0 {
			limit = limitNum
		}
	}

	books, err := h.bookService.GetBooksByCategory(r.Context(), categoryID, page, limit)
	if err != nil {
		h.log.Error("Failed to get books by category", zap.Error(err), zap.String("category_id", categoryID))

		if err.Error() == constants.ErrInternalServer {
			utils.RespondWithError(w, http.StatusInternalServerError, constants.ErrInternalServer, nil)
		} else {
			utils.RespondWithError(w, http.StatusBadRequest, err.Error(), nil)
		}
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Books by category retrieved successfully", books)
}
