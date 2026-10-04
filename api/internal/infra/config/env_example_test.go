package config

import (
	"bufio"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	pkgenv "github.com/zgiai/luas/api/pkg/env"
)

var envExampleKey = regexp.MustCompile(`^#?\s*([A-Z][A-Z0-9_]*)=`)

// TestEnvExampleDocumentsEveryVariableTheConfigReads keeps api/.env.example a complete reference:
// every variable that Load reads must appear there, set or commented out.
func TestEnvExampleDocumentsEveryVariableTheConfigReads(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("LUAS_ENV_FILE", "")
	t.Setenv("DB_ENABLED", "false")
	t.Setenv("OPTIONAL_STARTERS", strings.Join(StarterNames(), ","))
	t.Setenv("OPERATOR_ALLOWED_ORIGINS", "http://127.0.0.1:4173")
	t.Setenv("WEBHOOK_ENCRYPTION_KEY", "env-example-test-webhook-key-0123456789abcdef")
	t.Setenv("ASSET_TRANSFER_SIGNING_KEY", "env-example-test-asset-signing-key-0123456789")
	t.Cleanup(func() { LoadFresh() })
	if _, err := Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	documented := map[string]bool{}
	file, err := os.Open("../../../.env.example")
	if err != nil {
		t.Fatalf("open api/.env.example: %v", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if match := envExampleKey.FindStringSubmatch(strings.TrimSpace(scanner.Text())); match != nil {
			documented[match[1]] = true
		}
	}

	ignored := []string{
		"LUAS_ENV_FILE",   // locates the configuration file rather than configuring the app
		"GIN_MODE",        // legacy alias read only when SERVER_MODE is unset
		"LOG_FILENAME",    // legacy alias read only when LOG_FILE is unset
		"JWT_SECRET",      // read only to reject the removed JWT authentication
		"JWT_EXPIRE_DAYS", // read only to reject the removed JWT authentication
	}
	var missing []string
	for _, key := range pkgenv.RequestedKeys() {
		if !documented[key] && !slices.Contains(ignored, key) {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("api/.env.example does not document %d variable(s) the config reads:\n%s",
			len(missing), strings.Join(missing, "\n"))
	}
}
