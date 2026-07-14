package errors

import "testing"

func TestDefaultConfigDoesNotExposeDebugDetails(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Debug || cfg.ShowStack {
		t.Fatalf("DefaultConfig() exposes debug details: debug=%v stack=%v", cfg.Debug, cfg.ShowStack)
	}
	if !cfg.LogErrors {
		t.Fatal("DefaultConfig() must keep error logging enabled")
	}
}
