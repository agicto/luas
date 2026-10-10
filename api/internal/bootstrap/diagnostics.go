package bootstrap

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/zgiai/luas/api/internal/infra/config"
)

// newDiagnosticsServer serves Go runtime profiles on SERVER_DIAGNOSTICS_ADDR, which configuration
// validation restricts to loopback. It uses its own mux, so nothing registered on
// http.DefaultServeMux is exposed, and returns nil when diagnostics are off.
func newDiagnosticsServer(cfg *config.Config) *http.Server {
	if cfg == nil || cfg.Server.DiagnosticsAddr == "" {
		return nil
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	return &http.Server{
		Addr:              cfg.Server.DiagnosticsAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		// CPU profiles and traces stream for the requested duration (30s by default).
		WriteTimeout: 2 * time.Minute,
	}
}

func startDiagnosticsServer(srv *http.Server) {
	if srv == nil {
		return
	}
	go func() {
		slog.Warn("diagnostics.started", "listen", "http://"+srv.Addr+"/debug/pprof/")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("diagnostics.failed", "err", err)
		}
	}()
}
