package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/infra/database"
	"github.com/zgiai/luas/api/internal/modules/trend"
)

func main() {
	var (
		once      = flag.Bool("once", false, "run one sync and exit")
		interval  = flag.Duration("interval", 10*time.Minute, "sync interval")
		sourceURL = flag.String("source", trend.DailyDevHighlightsURL, "hotspot source URL")
	)
	flag.Parse()

	applyLocalEnvDefaults()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	applyLocalDefaults(cfg)

	db, err := database.NewDB(cfg)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}

	service := trend.NewService(
		trend.NewRepository(db),
		trend.NewDailyDevFetcher(&http.Client{Timeout: 30 * time.Second}),
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	run := func() {
		started := time.Now()
		result, err := service.SyncDailyDevHighlights(ctx, *sourceURL)
		if err != nil {
			log.Printf("trend sync failed after %s: %v", time.Since(started).Round(time.Millisecond), err)
			return
		}
		log.Printf(
			"trend sync ok: fetched=%d upserted=%d inserted=%d evaluated=%d candidates=%d queued_score_jobs=%d elapsed=%s",
			result.Fetched,
			result.Upserted,
			result.Inserted,
			result.Evaluated,
			result.Candidates,
			result.EnqueuedScoreJob,
			time.Since(started).Round(time.Millisecond),
		)
	}

	run()
	if *once {
		return
	}

	if *interval <= 0 {
		*interval = 10 * time.Minute
	}
	log.Printf("trend sync scheduler started: interval=%s next_run_at=%s", interval.String(), trend.NextRunAfter(*interval).Format(time.RFC3339))

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("trend sync scheduler stopped")
			return
		case <-ticker.C:
			run()
		}
	}
}

func applyLocalEnvDefaults() {
	appEnv := strings.TrimSpace(os.Getenv("APP_ENV"))
	if appEnv != "" && !strings.EqualFold(appEnv, "development") {
		return
	}
	setDefaultEnv("APP_ENV", "development")
	setDefaultEnv("DB_DRIVER", "postgres")
	setDefaultEnv("DB_HOST", "localhost")
	setDefaultEnv("DB_PORT", "5432")
	setDefaultEnv("DB_NAME", "agi01_content_local")
	setDefaultEnv("DB_USERNAME", os.Getenv("USER"))
	setDefaultEnv("DB_PASSWORD", "local-dev-password")
	setDefaultEnv("DB_SSLMODE", "disable")
	setDefaultEnv("DB_LOG_LEVEL", "warn")
	setDefaultEnv("JWT_SECRET", "local_trend_sync_secret_at_least_32_chars")
}

func setDefaultEnv(key, value string) {
	if strings.TrimSpace(os.Getenv(key)) != "" || strings.TrimSpace(value) == "" {
		return
	}
	_ = os.Setenv(key, value)
}

func applyLocalDefaults(cfg *config.Config) {
	if cfg == nil || !strings.EqualFold(cfg.App.Env, "development") {
		return
	}
	if strings.TrimSpace(cfg.Database.Name) == "" {
		cfg.Database.Name = "agi01_content_local"
	}
	if strings.TrimSpace(cfg.Database.Username) == "" {
		cfg.Database.Username = os.Getenv("USER")
	}
	if strings.TrimSpace(cfg.Database.Timezone) == "" {
		cfg.Database.Timezone = "UTC"
	}
}
