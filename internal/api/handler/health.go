package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/your-org/datavault/internal/domain/port"
	"github.com/your-org/datavault/internal/version"
)

type componentStatus struct {
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

type healthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Time    string `json:"time"`
}

type readyResponse struct {
	Status     string                     `json:"status"`
	Version    string                     `json:"version"`
	Time       string                     `json:"time"`
	Components map[string]componentStatus `json:"components"`
}

// Health handles GET /health — liveness probe.
// Always returns 200. Reports version and server time.
func Health() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, healthResponse{
			Status:  "ok",
			Version: version.Version,
			Time:    time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// Ready handles GET /ready — readiness probe.
// Checks database and HSM connectivity.
// Returns 503 if any component is unavailable.
func Ready(dbPinger port.Pinger, hsmPinger port.Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		components := make(map[string]componentStatus, 2)
		allOK := true

		// Database check
		if err := dbPinger.Ping(ctx); err != nil {
			components["db"] = componentStatus{Status: "error", Detail: "unreachable"}
			allOK = false
		} else {
			components["db"] = componentStatus{Status: "ok"}
		}

		// HSM check
		if err := hsmPinger.Ping(ctx); err != nil {
			components["hsm"] = componentStatus{Status: "error", Detail: "unreachable"}
			allOK = false
		} else {
			components["hsm"] = componentStatus{Status: "ok"}
		}

		status := "ready"
		code := http.StatusOK
		if !allOK {
			status = "unavailable"
			code = http.StatusServiceUnavailable
		}

		writeJSON(w, code, readyResponse{
			Status:     status,
			Version:    version.Version,
			Time:       time.Now().UTC().Format(time.RFC3339),
			Components: components,
		})
	}
}
