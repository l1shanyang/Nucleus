package handler

import (
	"net/http"

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
		return err
	}

	WriteSuccess(w, http.StatusCreated, note)
	return nil
}

func (h *NoteHandler) List(w http.ResponseWriter, r *http.Request) error {
	pagination, err := ParsePagination(r)
	if err != nil {
		return err
	}

	notes, err := h.svc.List(r.Context(), int32(pagination.Limit), int32(pagination.Offset))
	if err != nil {
		return err
	}

	WriteList(w, notes, pagination)
	return nil
}
