// Package api wires the HTTP router and registers all handler routes.
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/your-org/datavault/internal/api/handler"
	"github.com/your-org/datavault/internal/api/middleware"
	"github.com/your-org/datavault/internal/domain/port"
	"github.com/your-org/datavault/internal/domain/service"
	"github.com/your-org/datavault/internal/health"
	"github.com/your-org/datavault/internal/logger"
)

// NewRouter builds and returns the fully-configured chi router.
func NewRouter(svc *service.Service, log *logger.Logger, collector *health.Collector, keyValidator port.KeyValidator) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(middleware.StructuredLogger(log))

	// Health / readiness (no auth required)
	r.Get("/health", handler.Health(collector))
	r.Get("/ready", handler.Ready(collector))

	// Protected API routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.APIKeyAuth(log, keyValidator))

		r.Post("/v1/encrypt", handler.Encrypt(svc, log))
		r.Post("/v1/decrypt", handler.Decrypt(svc, log))
		r.Get("/v1/search", handler.Search(svc, log))
		r.Post("/v1/rewrap-dek", handler.Rewrap(svc, log))
	})

	return r
}
