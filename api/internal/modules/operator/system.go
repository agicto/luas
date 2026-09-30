package operator

import (
	"context"
	"runtime"
	"runtime/debug"
	"slices"

	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/pkg/response"
)

type systemResponse struct {
	Version   string   `json:"version"`
	Revision  string   `json:"revision"`
	GoVersion string   `json:"go_version"`
	Starters  []string `json:"starters"`
	Database  string   `json:"database"`
}

// activeStarters lists the default starters followed by the selected optional starters.
func activeStarters(cfg *config.Config) []string {
	starters := []string{config.StarterAudit, config.StarterAPIKey, config.StarterUser}
	if cfg != nil {
		for _, name := range cfg.Starters.Optional {
			if !slices.Contains(starters, name) {
				starters = append(starters, name)
			}
		}
	}
	return starters
}

func buildMetadata() (version string, revision string) {
	version, revision = "unknown", ""
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return version, revision
	}
	version = info.Main.Version
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" && len(setting.Value) >= 12 {
			revision = setting.Value[:12]
		}
	}
	return version, revision
}

func (s *service) databaseStatus(ctx context.Context) string {
	if err := s.grants.ping(ctx); err != nil {
		return "unavailable"
	}
	return "ok"
}

// System serves GET /v1/operator/system.
func (h *Handler) System(c *gin.Context) {
	version, revision := buildMetadata()
	response.Success(c, systemResponse{
		Version:   version,
		Revision:  revision,
		GoVersion: runtime.Version(),
		Starters:  activeStarters(h.cfg),
		Database:  h.service.databaseStatus(c.Request.Context()),
	})
}
