package handler

import (
	"encoding/json"
	"net/http"

	"github.com/your-org/datavault/internal/domain/service"
	"github.com/your-org/datavault/internal/logger"
)

type decryptRequest struct {
	TenantID string `json:"tenantId"`
	RecordID string `json:"recordId"`
}

type decryptResponse struct {
	RecordID  string `json:"recordId"`
	Plaintext []byte `json:"plaintextBase64"` // base64-encoded in JSON
}

// Decrypt handles POST /v1/decrypt
func Decrypt(svc *service.Service, log *logger.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req decryptRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.RecordID == "" || req.TenantID == "" {
			writeError(w, http.StatusBadRequest, "tenantId and recordId are required")
			return
		}

		resp, err := svc.Decrypt(r.Context(), service.DecryptRequest{
			TenantID:  req.TenantID,
			RecordID:  req.RecordID,
			Actor:     actorFromCtx(r),
			IPAddress: r.RemoteAddr,
		})
		if err != nil {
			log.Error("decrypt failed", "record_id", req.RecordID, "error", err)
			writeError(w, http.StatusNotFound, "record not found or decryption failed")
			return
		}

		writeJSON(w, http.StatusOK, decryptResponse{
			RecordID:  req.RecordID,
			Plaintext: resp.Plaintext,
		})
	}
}
