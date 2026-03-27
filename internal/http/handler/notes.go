package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"nucleus/internal/db/sqlc"
)

type NoteQueries interface {
	CreateNote(ctx context.Context, arg sqlc.CreateNoteParams) (sqlc.Note, error)
	ListNotes(ctx context.Context, arg sqlc.ListNotesParams) ([]sqlc.Note, error)
}

type NoteHandler struct {
	queries NoteQueries
}

type createNoteRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

func NewNoteHandler(queries NoteQueries) *NoteHandler {
	return &NoteHandler{queries: queries}
}

func (h *NoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	req.Body = strings.TrimSpace(req.Body)
	if req.Title == "" || req.Body == "" {
		writeError(w, http.StatusBadRequest, "title and body are required")
		return
	}

	note, err := h.queries.CreateNote(r.Context(), sqlc.CreateNoteParams{
		Title: req.Title,
		Body:  req.Body,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create note")
		return
	}

	writeJSON(w, http.StatusCreated, note)
}

func (h *NoteHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, err := parseIntWithDefault(r, "limit", 20)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if limit > 100 {
		limit = 100
	}

	offset, err := parseIntWithDefault(r, "offset", 0)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	notes, err := h.queries.ListNotes(r.Context(), sqlc.ListNotesParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list notes")
		return
	}

	writeJSON(w, http.StatusOK, notes)
}

func parseIntWithDefault(r *http.Request, key string, defaultValue int) (int, error) {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return defaultValue, nil
	}

	v, err := strconv.Atoi(value)
	if err != nil || v < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer", key)
	}

	return v, nil
}
