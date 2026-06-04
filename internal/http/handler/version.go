package handler

import (
	"net/http"

	"nucleus/internal/version"
)

type VersionHandler struct{}

func NewVersionHandler() *VersionHandler {
	return &VersionHandler{}
}

func (h *VersionHandler) Show(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, version.Get())
}
