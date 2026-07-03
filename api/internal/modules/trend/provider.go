package trend

import (
	"net/http"
	"time"

	"github.com/google/wire"

	"github.com/zgiai/luas/api/internal/contracts"
)

// ProviderSet wires the trend pipeline starter.
var ProviderSet = wire.NewSet(
	NewRepository,
	NewHTTPClient,
	NewDailyDevFetcher,
	NewService,
	NewHandler,
)

// NewHTTPClient provides the bounded client used by daily.dev fetches.
func NewHTTPClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}

// NewStarterManifest describes how the trend starter participates in the default scaffold.
func NewStarterManifest(handler *Handler) contracts.StarterManifest {
	return contracts.NewStaticStarterManifest(
		"trend",
		contracts.WithStarterModule(handler),
		contracts.WithStarterMigrationNames("2026_07_02_000000_create_content_pipeline_tables"),
	)
}
