// Package version exposes the build-time version string.
// The value is injected via ldflags at build time:
//
//	go build -ldflags "-X github.com/your-org/datavault/internal/version.Version=v0.1.0"
//
// Falls back to "dev" when built without ldflags (local development).
package version

// Version is set at build time. Default is "dev".
var Version = "dev"
