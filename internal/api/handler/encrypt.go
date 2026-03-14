package handler

import (
	"encoding/json"
	"net/http"

	"github.com/your-org/datavault/internal/domain/service"
	"github.com/your-org/datavault/internal/logger"
)

type encryptRequest struct {
	TenantID     string   `json:"tenantId"`
	Plaintext    []byte   `json:"plaintextBase64"` // base64-encoded in JSON
	AAD          []byte   `json:"aad"`             // base64-encoded in JSON
	SearchFields []string `json:"searchFields"`    // plaintext values for HMAC indexing
}

type encryptResponse struct {
	RecordID string `json:"recordId"`
}

// Encrypt handles POST /v1/encrypt
func Encrypt(svc *service.Service, log *logger.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req encryptRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		resp, err := svc.Encrypt(r.Context(), service.EncryptRequest{
			TenantID:     req.TenantID,
			Plaintext:    req.Plaintext,
			AAD:          req.AAD,
			SearchFields: req.SearchFields,
			Actor:        actorFromCtx(r),
			IPAddress:    r.RemoteAddr,
		})
		if err != nil {
			log.Error("encrypt failed", "error", err)
			writeError(w, http.StatusInternalServerError, "encryption failed")
			return
		}

		writeJSON(w, http.StatusCreated, encryptResponse{RecordID: resp.RecordID})
	}
}
