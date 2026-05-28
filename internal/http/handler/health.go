package handler

import (
	"context"
	"net/http"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type HealthHandler struct {
	db Pinger
}

type healthResponse struct {
	Status string `json:"status"`
}

type readyResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

func NewHealthHandler(db Pinger) *HealthHandler {
	return &HealthHandler{db: db}
}

// Liveness — 进程是否存活，不检查任何外部依赖。
func (h *HealthHandler) Live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

// Readiness — 服务是否就绪，检查数据库连接。
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	dbStatus := "ok"
	if err := h.db.Ping(r.Context()); err != nil {
		dbStatus = "unavailable"
		writeJSON(w, http.StatusServiceUnavailable, readyResponse{
			Status:   "not_ready",
			Database: dbStatus,
		})
		return
	}

	writeJSON(w, http.StatusOK, readyResponse{
		Status:   "ok",
		Database: dbStatus,
	})
}
