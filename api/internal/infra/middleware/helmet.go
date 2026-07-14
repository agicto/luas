package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type helmetHeader struct {
	name  string
	value string
}

// HelmetConfig holds security headers configuration
type HelmetConfig struct {
	// XSSProtection sets X-XSS-Protection header
	// Default: "1; mode=block"
	XSSProtection string

	// ContentTypeNosniff sets X-Content-Type-Options header
	// Default: "nosniff"
	ContentTypeNosniff string

	// XFrameOptions sets X-Frame-Options header
	// Default: "SAMEORIGIN"
	XFrameOptions string

	// HSTSMaxAge sets Strict-Transport-Security max-age
	// Default: 31536000 (1 year)
	// Set to 0 to disable HSTS
	HSTSMaxAge int

	// HSTSIncludeSubdomains adds includeSubDomains to HSTS
	// Default: true
	HSTSIncludeSubdomains bool

	// HSTSPreload adds preload to HSTS
	// Default: false
	HSTSPreload bool

	// ContentSecurityPolicy sets Content-Security-Policy header
	// Default: "" (disabled)
	ContentSecurityPolicy string

	// ReferrerPolicy sets Referrer-Policy header
	// Default: "strict-origin-when-cross-origin"
	ReferrerPolicy string

	// PermissionsPolicy sets Permissions-Policy header
	// Default: "" (disabled)
	PermissionsPolicy string

	// CrossOriginEmbedderPolicy sets Cross-Origin-Embedder-Policy header
	// Default: "" (disabled)
	CrossOriginEmbedderPolicy string

	// CrossOriginOpenerPolicy sets Cross-Origin-Opener-Policy header
	// Default: "same-origin"
	CrossOriginOpenerPolicy string

	// CrossOriginResourcePolicy sets Cross-Origin-Resource-Policy header
	// Default: "same-origin"
	CrossOriginResourcePolicy string
}

// DefaultHelmetConfig returns default security headers configuration
func DefaultHelmetConfig() HelmetConfig {
	return HelmetConfig{
		XSSProtection:             "1; mode=block",
		ContentTypeNosniff:        "nosniff",
		XFrameOptions:             "SAMEORIGIN",
		HSTSMaxAge:                31536000,
		HSTSIncludeSubdomains:     true,
		HSTSPreload:               false,
		ReferrerPolicy:            "strict-origin-when-cross-origin",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "same-origin",
	}
}

// Helmet returns security headers middleware with default config
func Helmet() gin.HandlerFunc {
	return HelmetWithConfig(DefaultHelmetConfig())
}

// HelmetWithConfig returns security headers middleware with custom config
func HelmetWithConfig(cfg HelmetConfig) gin.HandlerFunc {
	headers := make([]helmetHeader, 0, 10)
	add := func(name, value string) {
		if value != "" {
			headers = append(headers, helmetHeader{name: http.CanonicalHeaderKey(name), value: value})
		}
	}

	add("X-XSS-Protection", cfg.XSSProtection)
	add("X-Content-Type-Options", cfg.ContentTypeNosniff)
	add("X-Frame-Options", cfg.XFrameOptions)

	if cfg.HSTSMaxAge > 0 {
		hsts := "max-age=" + strconv.Itoa(cfg.HSTSMaxAge)
		if cfg.HSTSIncludeSubdomains {
			hsts += "; includeSubDomains"
		}
		if cfg.HSTSPreload {
			hsts += "; preload"
		}
		add("Strict-Transport-Security", hsts)
	}

	add("Content-Security-Policy", cfg.ContentSecurityPolicy)
	add("Referrer-Policy", cfg.ReferrerPolicy)
	add("Permissions-Policy", cfg.PermissionsPolicy)
	add("Cross-Origin-Embedder-Policy", cfg.CrossOriginEmbedderPolicy)
	add("Cross-Origin-Opener-Policy", cfg.CrossOriginOpenerPolicy)
	add("Cross-Origin-Resource-Policy", cfg.CrossOriginResourcePolicy)

	return func(c *gin.Context) {
		for _, header := range headers {
			c.Header(header.name, header.value)
		}

		c.Next()
	}
}
