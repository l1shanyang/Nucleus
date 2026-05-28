package router

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"nucleus/internal/http/handler"
	"nucleus/internal/http/middleware"
)

func New(healthHandler *handler.HealthHandler, noteHandler *handler.NoteHandler) http.Handler {
	r := chi.NewRouter()

	// 全局中间件
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.CORS("*"))

	// 运维端点
	r.Get("/healthz", healthHandler.Live)
	r.Get("/readyz", healthHandler.Ready)

	// 业务 API
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/notes", handler.WrapHandler(noteHandler.Create))
		r.Get("/notes", handler.WrapHandler(noteHandler.List))
	})

	return r
}
