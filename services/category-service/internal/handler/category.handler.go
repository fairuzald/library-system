package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/fairuzald/library-system/pkg/constants"
	"github.com/fairuzald/library-system/pkg/logger"
	"github.com/fairuzald/library-system/pkg/middleware"
	"github.com/fairuzald/library-system/pkg/utils"
	"github.com/fairuzald/library-system/services/category-service/internal/entity/dto"
	"github.com/fairuzald/library-system/services/category-service/internal/service"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type CategoryHandler struct {
	categoryService service.CategoryService
	log             *logger.Logger
}

func NewCategoryHandler(categoryService service.CategoryService, log *logger.Logger) *CategoryHandler {
	return &CategoryHandler{
		categoryService: categoryService,
		log:             log,
	}
}

func (h *CategoryHandler) isAdminOrLibrarian(r *http.Request) bool {
	role, ok := r.Context().Value(middleware.UserRoleKey).(string)
	if !ok {
		return false
	}
	return role == constants.RoleAdmin || role == constants.RoleLibrarian
}

func (h *CategoryHandler) HandleCreateCategory(w http.ResponseWriter, r *http.Request) {
	if !h.isAdminOrLibrarian(r) {
		utils.RespondWithError(w, http.StatusForbidden, constants.ErrForbidden, nil)
		return
	}

	var req dto.CategoryCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("Failed to decode request body", zap.Error(err))
		utils.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidRequest, err)
		return
	}

	if validationErrors, err := utils.Validate(req); err != nil {
		h.log.Info("Validation failed for create category request", zap.Any("errors", validationErrors))
		utils.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidField, err)
		return
	}

	category, err := h.categoryService.CreateCategory(r.Context(), &req)
	if err != nil {
		h.log.Error("Failed to create category", zap.Error(err))

		if err.Error() == constants.ErrInternalServer {
			utils.RespondWithError(w, http.StatusInternalServerError, constants.ErrInternalServer, nil)
		} else {
			utils.RespondWithError(w, http.StatusBadRequest, err.Error(), nil)
		}
		return
	}

	utils.RespondWithSuccess(w, http.StatusCreated, "Category created successfully", category)
}

func (h *CategoryHandler) HandleGetCategory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid category ID", err)
		return
	}

	category, err := h.categoryService.GetCategoryByID(r.Context(), id)
	if err != nil {
		if err.Error() == constants.ErrCategoryNotFound {
			utils.RespondWithError(w, http.StatusNotFound, constants.ErrCategoryNotFound, nil)
			return
		}

		h.log.Error("Failed to get category", zap.Error(err), zap.String("id", id.String()))
		utils.RespondWithError(w, http.StatusInternalServerError, constants.ErrInternalServer, nil)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Category retrieved successfully", category)
}

func (h *CategoryHandler) HandleGetCategoryByName(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Name parameter is required", nil)
		return
	}

	category, err := h.categoryService.GetCategoryByName(r.Context(), name)
	if err != nil {
		if err.Error() == constants.ErrCategoryNotFound || err.Error() == "category with name "+name+" not found" {
			utils.RespondWithError(w, http.StatusNotFound, err.Error(), nil)
			return
		}

		h.log.Error("Failed to get category by name", zap.Error(err), zap.String("name", name))
		utils.RespondWithError(w, http.StatusInternalServerError, constants.ErrInternalServer, nil)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Category retrieved successfully", category)
}

func (h *CategoryHandler) HandleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	if !h.isAdminOrLibrarian(r) {
		utils.RespondWithError(w, http.StatusForbidden, constants.ErrForbidden, nil)
		return
	}

	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid category ID", err)
		return
	}

	var req dto.CategoryUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("Failed to decode request body", zap.Error(err))
		utils.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidRequest, err)
		return
	}

	if validationErrors, err := utils.Validate(req); err != nil {
		h.log.Info("Validation failed for update category request", zap.Any("errors", validationErrors))
		utils.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidField, err)
		return
	}

	category, err := h.categoryService.UpdateCategory(r.Context(), id, &req)
	if err != nil {
		if err.Error() == constants.ErrCategoryNotFound {
			utils.RespondWithError(w, http.StatusNotFound, constants.ErrCategoryNotFound, nil)
			return
		}

		h.log.Error("Failed to update category", zap.Error(err), zap.String("id", id.String()))

		if err.Error() == constants.ErrInternalServer {
			utils.RespondWithError(w, http.StatusInternalServerError, constants.ErrInternalServer, nil)
		} else {
			utils.RespondWithError(w, http.StatusBadRequest, err.Error(), nil)
		}
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Category updated successfully", category)
}

func (h *CategoryHandler) HandleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	if !h.isAdminOrLibrarian(r) {
		utils.RespondWithError(w, http.StatusForbidden, constants.ErrForbidden, nil)
		return
	}

	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid category ID", err)
		return
	}

	if err := h.categoryService.DeleteCategory(r.Context(), id); err != nil {
		if err.Error() == constants.ErrCategoryNotFound {
			utils.RespondWithError(w, http.StatusNotFound, constants.ErrCategoryNotFound, nil)
			return
		}

		h.log.Error("Failed to delete category", zap.Error(err), zap.String("id", id.String()))

		if err.Error() == constants.ErrInternalServer {
			utils.RespondWithError(w, http.StatusInternalServerError, constants.ErrInternalServer, nil)
		} else {
			utils.RespondWithError(w, http.StatusBadRequest, err.Error(), nil)
		}
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Category deleted successfully", nil)
}

func (h *CategoryHandler) HandleListCategories(w http.ResponseWriter, r *http.Request) {
	filter := &dto.CategoryFilter{
		SortBy: r.URL.Query().Get("sort_by"),
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

	if parentID := r.URL.Query().Get("parent_id"); parentID != "" {
		filter.ParentID = &parentID
	}

	categories, err := h.categoryService.ListCategories(r.Context(), filter)
	if err != nil {
		h.log.Error("Failed to list categories", zap.Error(err))
		utils.RespondWithError(w, http.StatusInternalServerError, constants.ErrInternalServer, nil)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Categories retrieved successfully", categories)
}

func (h *CategoryHandler) HandleGetCategoryChildren(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid category ID", err)
		return
	}

	children, err := h.categoryService.GetCategoryChildren(r.Context(), id)
	if err != nil {
		if err.Error() == constants.ErrCategoryNotFound {
			utils.RespondWithError(w, http.StatusNotFound, constants.ErrCategoryNotFound, nil)
			return
		}

		h.log.Error("Failed to get category children", zap.Error(err), zap.String("id", id.String()))
		utils.RespondWithError(w, http.StatusInternalServerError, constants.ErrInternalServer, nil)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Category children retrieved successfully", children)
}
