package middleware

import (
	"net/http"
	"strings"

	"github.com/your-org/datavault/internal/api/handler"
	"github.com/your-org/datavault/internal/domain/port"
	"github.com/your-org/datavault/internal/logger"
)

// APIKeyAuth validates the Bearer token in the Authorization header using the
// supplied KeyValidator. The raw token is never logged — only the resolved
// actor name is attached to the request context.
func APIKeyAuth(log *logger.Logger, validator port.KeyValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				log.Warn("missing or malformed Authorization header", "remote", r.RemoteAddr)
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			actor, err := validator.Validate(r.Context(), token)
			if err != nil {
				// Log only at Warn — never log the token itself.
				log.Warn("API key validation failed", "remote", r.RemoteAddr)
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			ctx := handler.WithActor(r.Context(), actor)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
