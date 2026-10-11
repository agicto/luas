package commands

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/fastagi/fastbuild/api/internal/bootstrap"
	"github.com/fastagi/fastbuild/api/internal/infra/config"
	"github.com/fastagi/fastbuild/api/internal/infra/console"
	"github.com/fastagi/fastbuild/api/internal/infra/database"
	"github.com/fastagi/fastbuild/api/internal/infra/ratelimit"
)

const (
	defaultRateLimitPruneBatch      = 1000
	maxRateLimitPruneBatch          = 10_000
	defaultRateLimitPruneMaxBatches = 100
	maxRateLimitPruneMaxBatches     = 1000
)

// RateLimitPruneCommand deletes expired shared rate-limit buckets in bounded batches.
type RateLimitPruneCommand struct {
	output *console.Output
}

func NewRateLimitPruneCommand() *RateLimitPruneCommand {
	return &RateLimitPruneCommand{output: console.NewOutput()}
}

func (c *RateLimitPruneCommand) Name() string { return "ratelimit:prune" }
func (c *RateLimitPruneCommand) Description() string {
	return "Delete expired shared rate-limit buckets (RATE_LIMIT_STORE=postgres)"
}
func (c *RateLimitPruneCommand) Usage() string {
	return "ratelimit:prune [--batch=1000] [--max-batches=100]"
}

func (c *RateLimitPruneCommand) Run(args []string) error {
	batch, maxBatches, err := parseRateLimitPruneArgs(args)
	if err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if loggerErr := bootstrap.InitLogger(cfg); loggerErr != nil {
		return loggerErr
	}
	db, err := database.NewDB(cfg)
	if err != nil {
		return err
	}
	if db == nil {
		return fmt.Errorf("ratelimit:prune requires DB_ENABLED=true")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	deleted, err := ratelimit.PruneExpiredBuckets(ctx, sqlDB, batch, maxBatches)
	if err != nil {
		return fmt.Errorf("prune rate-limit buckets after deleting %d: %w", deleted, err)
	}
	slog.InfoContext(ctx, "ratelimit.buckets_pruned", "deleted", deleted)
	c.output.Success("Pruned %d expired rate-limit bucket(s)", deleted)
	return nil
}

func parseRateLimitPruneArgs(args []string) (int, int, error) {
	for _, arg := range args {
		if !strings.HasPrefix(arg, "--batch=") && !strings.HasPrefix(arg, "--max-batches=") {
			return 0, 0, fmt.Errorf("unknown ratelimit:prune argument %q", arg)
		}
	}
	batch, err := intFlag(args, "batch", defaultRateLimitPruneBatch, maxRateLimitPruneBatch)
	if err != nil {
		return 0, 0, err
	}
	maxBatches, err := intFlag(args, "max-batches", defaultRateLimitPruneMaxBatches, maxRateLimitPruneMaxBatches)
	if err != nil {
		return 0, 0, err
	}
	return batch, maxBatches, nil
}
