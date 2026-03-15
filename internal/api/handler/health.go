package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/your-org/datavault/internal/health"
)

// Health handles GET /health — liveness probe.
// Always returns 200. No external I/O — safe under any load.
func Health(col *health.Collector) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, col.Health())
	}
}

// Ready handles GET /ready — readiness probe.
// Pings DB and HSM; returns 503 when any component is unavailable.
func Ready(col *health.Collector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		snap, ok := col.Ready(ctx)
		code := http.StatusOK
		if !ok {
			code = http.StatusServiceUnavailable
		}
		writeJSON(w, code, snap)
	}
}
