package handler

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/fairuzald/library-system/pkg/logger"
	"github.com/fairuzald/library-system/pkg/utils"
	"go.uber.org/zap"
)

type HealthHandler struct {
	db  *sql.DB
	log *logger.Logger
}

func NewHealthHandler(db *sql.DB, log *logger.Logger) *HealthHandler {
	return &HealthHandler{
		db:  db,
		log: log,
	}
}

type HealthResponse struct {
	Status    string            `json:"status"`
	Version   string            `json:"version"`
	Timestamp time.Time         `json:"timestamp"`
	Checks    map[string]string `json:"checks"`
}

func (h *HealthHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	response := HealthResponse{
		Status:    "ok",
		Version:   "1.0.0",
		Timestamp: time.Now(),
		Checks: map[string]string{
			"database": "ok",
		},
	}

	if err := h.db.PingContext(ctx); err != nil {
		h.log.Error("Database health check failed", zap.Error(err))
		response.Status = "error"
		response.Checks["database"] = "error: " + err.Error()
	}

	utils.RespondWithJSON(w, http.StatusOK, response)
}

func (h *HealthHandler) GRPCHealth(ctx context.Context) (*HealthResponse, error) {
	response := &HealthResponse{
		Status:    "ok",
		Version:   "1.0.0",
		Timestamp: time.Now(),
		Checks: map[string]string{
			"database": "ok",
		},
	}

	if err := h.db.PingContext(ctx); err != nil {
		h.log.Error("Database health check failed", zap.Error(err))
		response.Status = "error"
		response.Checks["database"] = "error: " + err.Error()
	}

	return response, nil
}
