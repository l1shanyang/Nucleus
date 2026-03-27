package handler

import "net/http"

type HealthHandler struct{}

type healthResponse struct {
	Status string `json:"status"`
}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Get(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{
		Status: "ok",
	})
}
