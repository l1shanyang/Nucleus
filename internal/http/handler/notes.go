package handler

import (
	"net/http"
	"strconv"
	"strings"

	"nucleus/internal/store"
)

type NoteHandler struct {
	store store.NoteStore
}

type createNoteRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

func NewNoteHandler(s store.NoteStore) *NoteHandler {
	return &NoteHandler{store: s}
}

func (h *NoteHandler) Create(w http.ResponseWriter, r *http.Request) error {
	var req createNoteRequest
	if err := DecodeJSON(r, &req); err != nil {
		return err
	}

	req.Title = TrimString(req.Title)
	req.Body = TrimString(req.Body)
	if err := Require(map[string]string{"title": req.Title, "body": req.Body}); err != nil {
		return err
	}

	note, err := h.store.Create(r.Context(), req.Title, req.Body)
	if err != nil {
		return Internal("failed to create note")
	}

	WriteSuccess(w, http.StatusCreated, note)
	return nil
}

func (h *NoteHandler) List(w http.ResponseWriter, r *http.Request) error {
	limit, err := parseIntWithDefault(r, "limit", 20)
	if err != nil {
		return BadRequest(err.Error())
	}
	if limit > 100 {
		limit = 100
	}

	offset, err := parseIntWithDefault(r, "offset", 0)
	if err != nil {
		return BadRequest(err.Error())
	}

	notes, err := h.store.List(r.Context(), int32(limit), int32(offset))
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
