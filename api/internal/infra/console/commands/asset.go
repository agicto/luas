package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"github.com/zgiai/luas/api/internal/bootstrap"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/infra/console"
	"github.com/zgiai/luas/api/internal/wiring"
)

const defaultAssetPruneBatch = 100

type assetMaintainer interface {
	Prune(context.Context, int) (int, error)
}

var errAssetMaintainerUnavailable = errors.New("asset maintainer is unavailable")

// AssetPruneCommand removes expired private staging objects in one bounded batch.
type AssetPruneCommand struct {
	output *console.Output
}

func NewAssetPruneCommand() *AssetPruneCommand {
	return &AssetPruneCommand{output: console.NewOutput()}
}

func (c *AssetPruneCommand) Name() string        { return "asset:prune" }
func (c *AssetPruneCommand) Description() string { return "Prune expired asset staging objects" }
func (c *AssetPruneCommand) Usage() string       { return "asset:prune [--batch=100]" }

func (c *AssetPruneCommand) Run(args []string) error {
	batch, err := parseAssetPruneArgs(args)
	if err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if !slices.Contains(cfg.Starters.Optional, "asset") {
		return fmt.Errorf("asset starter is not selected in OPTIONAL_STARTERS")
	}
	if loggerErr := bootstrap.InitLogger(cfg); loggerErr != nil {
		return loggerErr
	}
	application, err := wiring.InitApplicationWithConfig(cfg)
	if err != nil {
		return fmt.Errorf("initialize asset cleanup: %w", err)
	}
	if application.AssetMaintainer == nil {
		return errAssetMaintainerUnavailable
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	processed, err := runAssetPrune(ctx, application.AssetMaintainer, batch)
	if err != nil {
		return err
	}
	c.output.Success("Pruned %d asset records", processed)
	return nil
}

func runAssetPrune(ctx context.Context, maintainer assetMaintainer, batch int) (int, error) {
	if maintainer == nil {
		return 0, errAssetMaintainerUnavailable
	}
	return maintainer.Prune(ctx, batch)
}

func parseAssetPruneArgs(args []string) (int, error) {
	batch := defaultAssetPruneBatch
	for index := 0; index < len(args); index++ {
		argument := args[index]
		switch {
		case argument == "--batch" && index+1 < len(args):
			index++
			value, err := strconv.Atoi(args[index])
			if err != nil {
				return 0, fmt.Errorf("invalid --batch value %q: %w", args[index], err)
			}
			batch = value
		case strings.HasPrefix(argument, "--batch="):
			value, err := strconv.Atoi(strings.TrimPrefix(argument, "--batch="))
			if err != nil {
				return 0, fmt.Errorf("invalid --batch value: %w", err)
			}
			batch = value
		default:
			return 0, fmt.Errorf("unknown asset prune argument %q", argument)
		}
	}
	if batch < 1 || batch > 100 {
		return 0, fmt.Errorf("--batch must be between 1 and 100")
	}
	return batch, nil
}
