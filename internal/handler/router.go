package handler

import (
	"github.com/mcp-reg/mcp-sub-registry/internal/frontend"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// NewRouter creates and configures the chi router
func NewRouter(h *Handler) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Health endpoint
	r.Get("/health", h.Health)

	// Registry API routes only (SSR removed, React SPA handles UI)
	r.Route("/{org}/{repo}/{branch}", func(r chi.Router) {
		r.Route("/v0.1", func(r chi.Router) {
			r.Post("/refresh", h.RefreshCache)
			r.Get("/servers", h.ListServers)
			r.Route("/servers/{serverName}/versions", func(r chi.Router) {
				r.Get("/", h.ListVersions)
				r.Get("/latest", h.GetLatestVersion)
				r.Get("/{version}", h.GetSpecificVersion)
			})
		})
	})

	// Embedded frontend (only when built with -tags=embed)
	if frontendHandler := frontend.Handler(); frontendHandler != nil {
		r.Handle("/*", frontendHandler)
	}

	return r
}
