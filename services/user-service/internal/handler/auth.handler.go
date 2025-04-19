package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/fairuzald/library-system/pkg/constants"
	"github.com/fairuzald/library-system/pkg/logger"
	"github.com/fairuzald/library-system/pkg/utils"
	"github.com/fairuzald/library-system/services/user-service/internal/entity/dto"
	"github.com/fairuzald/library-system/services/user-service/internal/service"
	"go.uber.org/zap"
)

type AuthHandler struct {
	authService service.AuthService
	log         *logger.Logger
}

func NewAuthHandler(authService service.AuthService, log *logger.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		log:         log,
	}
}

func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req dto.UserLogin

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("Failed to decode login request", zap.Error(err))
		utils.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidRequest, err)
		return
	}

	if validationErrors, err := utils.Validate(req); err != nil {
		h.log.Info("Validation failed for login request", zap.Any("errors", validationErrors))
		utils.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidField, err)
		return
	}

	response, err := h.authService.Login(r.Context(), &req)
	if err != nil {
		if err.Error() == constants.ErrInvalidCredentials {
			utils.RespondWithError(w, http.StatusUnauthorized, constants.ErrInvalidCredentials, nil)
			return
		}

		h.log.Error("Login failed", zap.Error(err))
		utils.RespondWithError(w, http.StatusInternalServerError, constants.ErrInternalServer, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Login successful", response)
}

func (h *AuthHandler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req dto.UserRegister

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("Failed to decode register request", zap.Error(err))
		utils.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidRequest, err)
		return
	}

	if validationErrors, err := utils.Validate(req); err != nil {
		h.log.Info("Validation failed for register request", zap.Any("errors", validationErrors))
		utils.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidField, err)
		return
	}

	user, err := h.authService.Register(r.Context(), &req)
	if err != nil {
		if err.Error() == constants.ErrEmailTaken || err.Error() == constants.ErrUsernameTaken {
			utils.RespondWithError(w, http.StatusConflict, err.Error(), nil)
			return
		}

		h.log.Error("Failed to register user", zap.Error(err))
		utils.RespondWithError(w, http.StatusInternalServerError, constants.ErrInternalServer, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusCreated, "User registered successfully", user)
}

func (h *AuthHandler) HandleRefreshToken(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshToken

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("Failed to decode refresh token request", zap.Error(err))
		utils.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidRequest, err)
		return
	}

	if validationErrors, err := utils.Validate(req); err != nil {
		h.log.Info("Validation failed for refresh token request", zap.Any("errors", validationErrors))
		utils.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidField, err)
		return
	}

	response, err := h.authService.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		if err.Error() == constants.ErrInvalidToken {
			utils.RespondWithError(w, http.StatusUnauthorized, constants.ErrInvalidToken, nil)
			return
		}

		h.log.Error("Token refresh failed", zap.Error(err))
		utils.RespondWithError(w, http.StatusInternalServerError, constants.ErrInternalServer, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Token refreshed successfully", response)
}

func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get(constants.HeaderAuthorization)
	if authHeader == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "No authorization header provided", nil)
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != constants.TokenTypBearer {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid authorization header format", nil)
		return
	}

	accessToken := parts[1]

	if err := h.authService.Logout(r.Context(), accessToken); err != nil {
		if err.Error() == constants.ErrInvalidToken {
			utils.RespondWithError(w, http.StatusUnauthorized, constants.ErrInvalidToken, nil)
			return
		}

		h.log.Error("Logout failed", zap.Error(err))
		utils.RespondWithError(w, http.StatusInternalServerError, constants.ErrInternalServer, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Logout successful", nil)
}
