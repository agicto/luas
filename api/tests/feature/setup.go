package feature

import (
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/bootstrap"
	"github.com/zgiai/luas/api/internal/infra/config"
	test_platform "github.com/zgiai/luas/api/internal/infra/testing"
	"github.com/zgiai/luas/api/internal/wiring"
)

// SetupApp initializes the feature-test application by reusing the production DI graph
// and HTTP startup chain with a test-specific config.
func SetupApp(t *testing.T) *gin.Engine {
	t.Helper()
	return SetupAppWithOptionalStarters(t)
}

// SetupAppWithOptionalStarters reuses production assembly with an explicit additive selection.
func SetupAppWithOptionalStarters(t *testing.T, optionalStarters ...string) *gin.Engine {
	t.Helper()
	return setupApp(t, nil, optionalStarters...)
}

func setupApp(t *testing.T, configure func(*config.Config), optionalStarters ...string) *gin.Engine {
	t.Helper()
	cfg := &config.Config{}
	cfg.App.Name = "Luas Test"
	cfg.App.Env = "test"
	cfg.App.Debug = false
	cfg.App.URL = "http://localhost:0"
	cfg.Starters.Optional = append([]string(nil), optionalStarters...)
	cfg.Server.Mode = "test"
	cfg.Database = test_platform.CreatePostgresDatabase(t)
	cfg.Queue = config.QueueConfig{
		Driver: "sync", DefaultQueue: "default", BufferSize: 256,
		WorkerConcurrency: 1, WorkerSleep: time.Second, WorkerTimeout: time.Minute,
		LeaseDuration: 90 * time.Second, HeartbeatInterval: 20 * time.Second,
	}
	cfg.Authentication.SessionTTL = config.DefaultAuthenticationSessionTTL
	cfg.Authentication.SessionIdleTimeout = config.DefaultAuthenticationSessionIdleTimeout
	cfg.Authentication.SessionTouchInterval = config.DefaultAuthenticationSessionTouchInterval
	cfg.Authentication.SessionRetention = config.DefaultAuthenticationSessionRetention
	cfg.CORS.AllowOrigins = []string{"http://localhost:3000"}
	cfg.CORS.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	cfg.CORS.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "Organization-Id", "X-API-Key", "X-Request-ID"}
	cfg.CORS.ExposeHeaders = []string{"Content-Length", "X-Request-ID"}
	cfg.CORS.AllowCredentials = true
	cfg.AI.DefaultProvider = "openai"
	cfg.AI.DefaultModel = "gpt-5"
	cfg.Organization.InvitationTTL = config.DefaultOrganizationInvitationTTL
	if configure != nil {
		configure(cfg)
	}

	if _, err := config.Use(cfg); err != nil {
		panic("failed to register test config: " + err.Error())
	}

	application, err := wiring.InitApplicationWithConfig(cfg)
	if err != nil {
		panic("failed to init test application: " + err.Error())
	}
	if sqlDB, dbErr := application.DB.DB(); dbErr == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}

	if err := bootstrap.RunRegisteredMigrations(application.Migrator); err != nil {
		panic("failed to run migrations for test db: " + err.Error())
	}

	kernel := bootstrap.NewHttpKernel(application)
	return kernel.Engine
}

// NewTestCase is a shortcut to create a test case with the setup app
func NewTestCase(t *testing.T) *test_platform.TestCase {
	engine := SetupApp(t)
	return test_platform.NewTestCase(t, engine)
}

// NewTestCaseWithOptionalStarters creates a feature test with additive starters enabled.
func NewTestCaseWithOptionalStarters(t *testing.T, optionalStarters ...string) *test_platform.TestCase {
	engine := SetupAppWithOptionalStarters(t, optionalStarters...)
	return test_platform.NewTestCase(t, engine)
}
