package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func realIPForRequest(t *testing.T, cfg RealIPConfig, remoteAddr string, headers map[string]string) string {
	t.Helper()

	engine := gin.New()
	if err := engine.SetTrustedProxies(nil); err != nil {
		t.Fatalf("SetTrustedProxies(nil) error = %v", err)
	}
	engine.Use(RealIPWithConfig(cfg))
	engine.GET("/ip", func(c *gin.Context) {
		c.String(http.StatusOK, GetRealIP(c))
	})

	request := httptest.NewRequest(http.MethodGet, "/ip", nil)
	request.RemoteAddr = remoteAddr
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	return response.Body.String()
}

func TestRealIPDoesNotTrustForwardingHeadersByDefault(t *testing.T) {
	got := realIPForRequest(t, DefaultRealIPConfig(), "198.51.100.10:1234", map[string]string{
		HeaderXForwardedFor: "203.0.113.20",
		HeaderXRealIP:       "203.0.113.21",
	})
	if got != "198.51.100.10" {
		t.Fatalf("real IP = %q, want direct peer", got)
	}
}

func TestRealIPAcceptsForwardingHeaderFromTrustedProxy(t *testing.T) {
	cfg := DefaultRealIPConfig()
	cfg.TrustedProxies = []string{"198.51.100.10"}

	got := realIPForRequest(t, cfg, "198.51.100.10:1234", map[string]string{
		HeaderXForwardedFor: "203.0.113.20",
	})
	if got != "203.0.113.20" {
		t.Fatalf("real IP = %q, want forwarded client", got)
	}
}

func TestRealIPWalksTrustedForwardingChainFromRight(t *testing.T) {
	cfg := DefaultRealIPConfig()
	cfg.TrustedProxies = []string{"10.0.0.0/8"}

	got := realIPForRequest(t, cfg, "10.0.0.2:1234", map[string]string{
		HeaderXForwardedFor: "203.0.113.20, 10.0.0.1",
	})
	if got != "203.0.113.20" {
		t.Fatalf("real IP = %q, want first untrusted address from right", got)
	}
}

func TestRealIPIgnoresInvalidForwardedAddress(t *testing.T) {
	cfg := DefaultRealIPConfig()
	cfg.TrustedProxies = []string{"198.51.100.10"}

	got := realIPForRequest(t, cfg, "198.51.100.10:1234", map[string]string{
		HeaderXForwardedFor: "not-an-ip",
	})
	if got != "198.51.100.10" {
		t.Fatalf("real IP = %q, want direct peer", got)
	}
}

func TestRealIPIgnoresTrustAllProxyRanges(t *testing.T) {
	cfg := DefaultRealIPConfig()
	cfg.TrustedProxies = []string{"0.0.0.0/0", "::/0"}

	got := realIPForRequest(t, cfg, "198.51.100.10:1234", map[string]string{
		HeaderXForwardedFor: "203.0.113.20",
	})
	if got != "198.51.100.10" {
		t.Fatalf("real IP = %q, want direct peer when trust-all ranges are ignored", got)
	}
}
