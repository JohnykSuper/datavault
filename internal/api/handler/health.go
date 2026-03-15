package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/your-org/datavault/internal/health"
)

// Health handles GET /health — unified liveness + readiness + telemetry probe.
// Returns 200 with status "ok" when all components are healthy.
// Returns 503 with status "error" when any component is unavailable.
func Health(col *health.Collector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		snap, ok := col.Check(ctx)
		code := http.StatusOK
		if !ok {
			code = http.StatusServiceUnavailable
		}
		writeJSON(w, code, snap)
	}
}
