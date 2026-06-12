package router

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"nucleus/internal/http/handler"
	"nucleus/internal/http/middleware"
)

type Options struct {
	CORSAllowedOrigins []string
}

func New(
	healthHandler *handler.HealthHandler,
	versionHandler *handler.VersionHandler,
	authHandler *handler.AuthHandler,
	workspaceHandler *handler.WorkspaceHandler,
	noteHandler *handler.NoteHandler,
	opts Options,
) http.Handler {
	r := chi.NewRouter()

	// 全局中间件
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.RequestLog)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.CORS(opts.CORSAllowedOrigins...))

	// 运维端点
	r.Get("/healthz", healthHandler.Live)
	r.Get("/readyz", healthHandler.Ready)
	r.Get("/version", versionHandler.Show)

	// 业务 API
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/register", handler.WrapHandler(authHandler.Register))
		r.Post("/auth/login", handler.WrapHandler(authHandler.Login))
		r.With(authHandler.RequireAuth).Get("/me", handler.WrapHandler(authHandler.Me))
		r.Group(func(r chi.Router) {
			r.Use(authHandler.RequireAuth)
			r.Post("/workspaces", handler.WrapHandler(workspaceHandler.Create))
			r.Get("/workspaces", handler.WrapHandler(workspaceHandler.List))
			r.Get("/workspaces/{workspaceID}", handler.WrapHandler(workspaceHandler.Get))
			r.Post("/workspaces/{workspaceID}/members", handler.WrapHandler(workspaceHandler.AddMember))
			r.Get("/workspaces/{workspaceID}/members", handler.WrapHandler(workspaceHandler.ListMembers))
			r.Patch("/workspaces/{workspaceID}/members/{memberID}/role", handler.WrapHandler(workspaceHandler.UpdateMemberRole))
			r.Delete("/workspaces/{workspaceID}/members/{memberID}", handler.WrapHandler(workspaceHandler.RemoveMember))
		})
		r.Post("/notes", handler.WrapHandler(noteHandler.Create))
		r.Get("/notes", handler.WrapHandler(noteHandler.List))
	})

	return r
}
