package logger

// Boot initializes the standalone logger with library defaults. Application
// runtimes should map their typed configuration and call BootWithConfig.
func Boot() *Logger {
	return BootWithConfig(DefaultConfig())
}

// BootWithConfig installs an explicitly configured process logger. It does
// not read environment variables; application configuration remains owned by
// the bootstrap layer.
func BootWithConfig(cfg *Config) *Logger {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	runtimeLogger := New(cfg)
	SetDefault(runtimeLogger)
	return runtimeLogger
}
