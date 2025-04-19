package handler

import (
	"encoding/json"
	"net/http"

	"github.com/fairuzald/library-system/pkg/constants"
	"github.com/fairuzald/library-system/pkg/logger"
	"github.com/fairuzald/library-system/pkg/middleware"
	"github.com/fairuzald/library-system/pkg/utils"
	"github.com/fairuzald/library-system/services/user-service/internal/entity/dto"
	"github.com/fairuzald/library-system/services/user-service/internal/service"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type UserHandler struct {
	userService service.UserService
	log         *logger.Logger
}

func (h *UserHandler) isAdmin(r *http.Request) bool {
	role, ok := r.Context().Value(middleware.UserRoleKey).(string)
	if !ok {
		return false
	}
	return role == constants.RoleAdmin
}

func (h *UserHandler) isSameUser(r *http.Request, id uuid.UUID) bool {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		return false
	}
	return userID == id.String()
}

func (h *UserHandler) isAdminOrSameUser(r *http.Request, id uuid.UUID) bool {
	return h.isAdmin(r) || h.isSameUser(r, id)
}

func (h *UserHandler) isLibrarianOrAbove(r *http.Request) bool {
	role, ok := r.Context().Value(middleware.UserRoleKey).(string)
	if !ok {
		return false
	}
	return role == constants.RoleAdmin || role == constants.RoleLibrarian
}

func NewUserHandler(userService service.UserService, log *logger.Logger) *UserHandler {
	return &UserHandler{
		userService: userService,
		log:         log,
	}
}

func (h *UserHandler) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req dto.UserCreate

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("Failed to decode request body", zap.Error(err))
		utils.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidRequest, err)
		return
	}

	if validationErrors, err := utils.Validate(req); err != nil {
		h.log.Info("Validation failed for create user request", zap.Any("errors", validationErrors))
		utils.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidField, err)
		return
	}

	user, err := h.userService.CreateUser(r.Context(), &req)
	if err != nil {
		if err.Error() == constants.ErrEmailTaken || err.Error() == constants.ErrUsernameTaken {
			utils.RespondWithError(w, http.StatusConflict, err.Error(), nil)
			return
		}

		h.log.Error("Failed to create user", zap.Error(err))
		utils.RespondWithError(w, http.StatusInternalServerError, constants.ErrInternalServer, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusCreated, "User created successfully", user)
}

func (h *UserHandler) HandleGetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid user ID", err)
		return
	}

	user, err := h.userService.GetUserByID(r.Context(), id)
	if err != nil {
		if err.Error() == constants.ErrUserNotFound {
			utils.RespondWithError(w, http.StatusNotFound, constants.ErrUserNotFound, nil)
			return
		}

		h.log.Error("Failed to get user", zap.Error(err), zap.String("id", id.String()))
		utils.RespondWithError(w, http.StatusInternalServerError, constants.ErrInternalServer, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "User retrieved successfully", user)
}

func (h *UserHandler) HandleGetUserByEmail(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Email parameter is required", nil)
		return
	}

	user, err := h.userService.GetUserByEmail(r.Context(), email)
	if err != nil {
		if err.Error() == constants.ErrUserNotFound {
			utils.RespondWithError(w, http.StatusNotFound, constants.ErrUserNotFound, nil)
			return
		}

		h.log.Error("Failed to get user by email", zap.Error(err), zap.String("email", email))
		utils.RespondWithError(w, http.StatusInternalServerError, constants.ErrInternalServer, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "User retrieved successfully", user)
}

func (h *UserHandler) HandleGetUserByUsername(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Username parameter is required", nil)
		return
	}

	user, err := h.userService.GetUserByUsername(r.Context(), username)
	if err != nil {
		if err.Error() == constants.ErrUserNotFound {
			utils.RespondWithError(w, http.StatusNotFound, constants.ErrUserNotFound, nil)
			return
		}

		h.log.Error("Failed to get user by username", zap.Error(err), zap.String("username", username))
		utils.RespondWithError(w, http.StatusInternalServerError, constants.ErrInternalServer, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "User retrieved successfully", user)
}

func (h *UserHandler) HandleUpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid user ID", err)
		return
	}

	var req dto.UserUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("Failed to decode request body", zap.Error(err))
		utils.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidRequest, err)
		return
	}

	if validationErrors, err := utils.Validate(req); err != nil {
		h.log.Info("Validation failed for update user request", zap.Any("errors", validationErrors))
		utils.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidField, err)
		return
	}

	if !h.isAdminOrSameUser(r, id) {
		utils.RespondWithError(w, http.StatusForbidden, constants.ErrForbidden, nil)
		return
	}

	user, err := h.userService.UpdateUser(r.Context(), id, &req)
	if err != nil {
		if err.Error() == constants.ErrUserNotFound {
			utils.RespondWithError(w, http.StatusNotFound, constants.ErrUserNotFound, nil)
			return
		}

		if err.Error() == constants.ErrEmailTaken || err.Error() == constants.ErrUsernameTaken {
			utils.RespondWithError(w, http.StatusConflict, err.Error(), nil)
			return
		}

		h.log.Error("Failed to update user", zap.Error(err), zap.String("id", id.String()))
		utils.RespondWithError(w, http.StatusInternalServerError, constants.ErrInternalServer, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "User updated successfully", user)
}

func (h *UserHandler) HandleDeleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid user ID", err)
		return
	}

	if !h.isAdmin(r) {
		utils.RespondWithError(w, http.StatusForbidden, constants.ErrForbidden, nil)
		return
	}

	if err := h.userService.DeleteUser(r.Context(), id); err != nil {
		if err.Error() == constants.ErrUserNotFound {
			utils.RespondWithError(w, http.StatusNotFound, constants.ErrUserNotFound, nil)
			return
		}

		h.log.Error("Failed to delete user", zap.Error(err), zap.String("id", id.String()))
		utils.RespondWithError(w, http.StatusInternalServerError, constants.ErrInternalServer, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "User deleted successfully", nil)
}

func (h *UserHandler) HandleListUsers(w http.ResponseWriter, r *http.Request) {
	filter := &dto.UserFilter{
		Role:   r.URL.Query().Get("role"),
		Status: r.URL.Query().Get("status"),
		SortBy: r.URL.Query().Get("sort_by"),
	}

	if page := r.URL.Query().Get("page"); page != "" {
		if pageNum, err := utils.ParseInt(page); err == nil {
			filter.Page = pageNum
		}
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if limitNum, err := utils.ParseInt(limit); err == nil {
			filter.Limit = limitNum
		}
	}

	if desc := r.URL.Query().Get("desc"); desc == "true" {
		filter.Desc = true
	}

	if !h.isAdmin(r) {
		utils.RespondWithError(w, http.StatusForbidden, constants.ErrForbidden, nil)
		return
	}

	response, err := h.userService.ListUsers(r.Context(), filter)
	if err != nil {
		h.log.Error("Failed to list users", zap.Error(err))
		utils.RespondWithError(w, http.StatusInternalServerError, constants.ErrInternalServer, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Users retrieved successfully", response)
}

func (h *UserHandler) HandleChangePassword(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid user ID", err)
		return
	}

	var req dto.ChangePassword
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("Failed to decode request body", zap.Error(err))
		utils.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidRequest, err)
		return
	}

	if validationErrors, err := utils.Validate(req); err != nil {
		h.log.Info("Validation failed for change password request", zap.Any("errors", validationErrors))
		utils.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidField, err)
		return
	}

	if !h.isSameUser(r, id) {
		utils.RespondWithError(w, http.StatusForbidden, constants.ErrForbidden, nil)
		return
	}

	if err := h.userService.ChangePassword(r.Context(), id, &req); err != nil {
		if err.Error() == constants.ErrUserNotFound {
			utils.RespondWithError(w, http.StatusNotFound, constants.ErrUserNotFound, nil)
			return
		}

		if err.Error() == constants.ErrInvalidCredentials {
			utils.RespondWithError(w, http.StatusUnauthorized, constants.ErrInvalidCredentials, nil)
			return
		}

		if err.Error() == constants.ErrWeakPassword {
			utils.RespondWithError(w, http.StatusBadRequest, constants.ErrWeakPassword, nil)
			return
		}

		h.log.Error("Failed to change password", zap.Error(err), zap.String("id", id.String()))
		utils.RespondWithError(w, http.StatusInternalServerError, constants.ErrInternalServer, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Password changed successfully", nil)
}
