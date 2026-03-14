package handler

import (
	"net/http"

	"github.com/your-org/datavault/internal/domain/service"
	"github.com/your-org/datavault/internal/logger"
)

type searchResponse struct {
	RecordIDs []string `json:"recordIds"`
}

// Search handles GET /v1/search?field=<plaintext_value>
// The plaintext field value is HMAC-hashed server-side and never logged.
func Search(svc *service.Service, log *logger.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.Header.Get("X-Tenant-ID")
		fieldValue := r.URL.Query().Get("field")

		if fieldValue == "" {
			writeError(w, http.StatusBadRequest, "query parameter 'field' is required")
			return
		}

		resp, err := svc.Search(r.Context(), service.SearchRequest{
			TenantID:   tenantID,
			FieldValue: fieldValue,
			Actor:      actorFromCtx(r),
			IPAddress:  r.RemoteAddr,
		})
		if err != nil {
			log.Error("search failed", "error", err)
			writeError(w, http.StatusInternalServerError, "search failed")
			return
		}

		writeJSON(w, http.StatusOK, searchResponse{RecordIDs: resp.RecordIDs})
	}
}
