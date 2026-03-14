package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/your-org/datavault/internal/domain/port"
)

// Health handles GET /health — always returns 200 (liveness probe).
func Health() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// Ready handles GET /ready — readiness probe.
// Returns 503 if the database cannot be reached within 3 seconds.
func Ready(pinger port.Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		if err := pinger.Ping(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status": "unavailable",
				"detail": "database unreachable",
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}
