package router

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"nucleus/internal/http/handler"
)

func New(healthHandler *handler.HealthHandler, noteHandler *handler.NoteHandler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/healthz", healthHandler.Get)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/notes", noteHandler.Create)
		r.Get("/notes", noteHandler.List)
	})

	return r
}
