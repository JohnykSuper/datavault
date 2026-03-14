package port

import "context"

// KeyValidator authenticates an incoming bearer token and returns the
// resolved actor identity (e.g. service name or tenant). Implementations
// may use a static config, a DB lookup, or an external secret manager.
type KeyValidator interface {
	Validate(ctx context.Context, token string) (actor string, err error)
}
