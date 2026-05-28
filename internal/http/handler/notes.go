package handler

import (
	"net/http"
	"strconv"
	"strings"

	"nucleus/internal/service"
)

type NoteHandler struct {
	svc *service.NoteService
}

type createNoteRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

func NewNoteHandler(svc *service.NoteService) *NoteHandler {
	return &NoteHandler{svc: svc}
}

func (h *NoteHandler) Create(w http.ResponseWriter, r *http.Request) error {
	var req createNoteRequest
	if err := DecodeJSON(r, &req); err != nil {
		return err
	}

	note, err := h.svc.Create(r.Context(), service.CreateInput{
		Title: req.Title,
		Body:  req.Body,
	})
	if err != nil {
		return BadRequest(err.Error())
	}

	WriteSuccess(w, http.StatusCreated, note)
	return nil
}

func (h *NoteHandler) List(w http.ResponseWriter, r *http.Request) error {
	limit, err := parseIntWithDefault(r, "limit", 20)
	if err != nil {
		return BadRequest(err.Error())
	}

	offset, err := parseIntWithDefault(r, "offset", 0)
	if err != nil {
		return BadRequest(err.Error())
	}

	notes, err := h.svc.List(r.Context(), int32(limit), int32(offset))
	if err != nil {
		return Internal("failed to list notes")
	}

	WriteList(w, notes, map[string]any{
		"limit":  limit,
		"offset": offset,
	})
	return nil
}

func parseIntWithDefault(r *http.Request, key string, defaultValue int) (int, error) {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return defaultValue, nil
	}

	v, err := strconv.Atoi(value)
	if err != nil || v < 0 {
		return 0, &AppError{
			Status:  http.StatusBadRequest,
			Code:    "INVALID_PARAM",
			Message: key + " must be a non-negative integer",
		}
	}

	return v, nil
}
