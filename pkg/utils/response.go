package utils

import (
	"encoding/json"
	"net/http"

	"github.com/fairuzald/library-system/pkg/models"
)

// RespondWithJSON writes a JSON response
func RespondWithJSON(w http.ResponseWriter, status int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error marshaling JSON"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(response)
}

// RespondWithError writes an error response
func RespondWithError(w http.ResponseWriter, status int, message string, err error) {
	errorResponse := models.ErrorResponse{
		Status:  status,
		Message: message,
	}

	if err != nil {
		errorResponse.Error = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorResponse)
}

// RespondWithSuccess writes a success response
func RespondWithSuccess(w http.ResponseWriter, status int, message string, data interface{}) {
	successResponse := models.SuccessResponse{
		Status:  status,
		Message: message,
		Data:    data,
	}

	RespondWithJSON(w, status, successResponse)
}

// RespondWithPagination writes a paginated response
func RespondWithPagination(w http.ResponseWriter, status int, data interface{}, totalItems int64, page, pageSize int) {
	totalPages := int(totalItems) / pageSize
	if int(totalItems)%pageSize > 0 {
		totalPages++
	}

	paginatedResponse := models.PaginatedResponse{
		TotalItems:  totalItems,
		TotalPages:  totalPages,
		CurrentPage: page,
		PageSize:    pageSize,
		Data:        data,
	}

	RespondWithJSON(w, status, paginatedResponse)
}
