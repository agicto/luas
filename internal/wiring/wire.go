//go:build wireinject
// +build wireinject

package wiring

import (
	"github.com/google/wire"
	"github.com/zgiai/zgo/internal/app"
	"github.com/zgiai/zgo/internal/infra"
	"github.com/zgiai/zgo/internal/infra/config"
	"github.com/zgiai/zgo/internal/starter"
)

// InitApplication initializes the entire application with all dependencies.
// This is the single entry point for Wire DI.
func InitApplication() (*app.Application, error) {
	wire.Build(
		// Infrastructure providers
		infra.ProviderSet,

		// Default starter providers
		starter.ProviderSet,

		// Build final application
		wire.Struct(new(app.Application), "*"),
	)
	return nil, nil
}

// InitApplicationWithConfig initializes the application using an explicitly supplied config.
// This is primarily used by tests so they can reuse the production DI graph and startup chain.
func InitApplicationWithConfig(cfg *config.Config) (*app.Application, error) {
	wire.Build(
		infra.ConfiguredProviderSet,
		starter.ProviderSet,
		wire.Struct(new(app.Application), "*"),
	)
	return nil, nil
}
