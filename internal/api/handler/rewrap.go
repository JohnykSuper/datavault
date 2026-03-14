package handler

import (
	"encoding/json"
	"net/http"

	"github.com/your-org/datavault/internal/domain/service"
	"github.com/your-org/datavault/internal/logger"
)

type rewrapRequest struct {
	TenantID   string `json:"tenantId"`
	OldVersion int    `json:"keyVersion"`
}

type rewrapResponse struct {
	Migrated int `json:"migrated"`
}

// Rewrap handles POST /v1/rewrap-dek
func Rewrap(svc *service.Service, log *logger.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req rewrapRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		resp, err := svc.RewrapDEK(r.Context(), service.RewrapRequest{
			TenantID:   req.TenantID,
			OldVersion: req.OldVersion,
			Actor:      actorFromCtx(r),
			IPAddress:  r.RemoteAddr,
		})
		if err != nil {
			log.Error("rewrap failed", "error", err)
			writeError(w, http.StatusInternalServerError, "rewrap failed")
			return
		}

		writeJSON(w, http.StatusOK, rewrapResponse{Migrated: resp.Migrated})
	}
}
