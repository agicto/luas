package operatorcommands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/zgiai/luas/api/internal/bootstrap"
	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/infra/console"
	"github.com/zgiai/luas/api/internal/wiring"
)

const (
	defaultAuthenticationSessionPruneBatch = 500
	maxAuthenticationSessionPruneBatch     = 10_000
)

var errAuthenticationSessionMaintainerUnavailable = errors.New("authentication session maintainer is unavailable")

// AuthenticationSessionPruneCommand removes one bounded batch of terminal sessions beyond retention.
type AuthenticationSessionPruneCommand struct {
	output *console.Output
}

func NewAuthenticationSessionPruneCommand() *AuthenticationSessionPruneCommand {
	return &AuthenticationSessionPruneCommand{output: console.NewOutput()}
}

func (c *AuthenticationSessionPruneCommand) Name() string { return "auth-session:prune" }
func (c *AuthenticationSessionPruneCommand) Description() string {
	return "Prune retained terminal authentication sessions"
}
func (c *AuthenticationSessionPruneCommand) Usage() string {
	return "auth-session:prune [--batch=500]"
}

func (c *AuthenticationSessionPruneCommand) Run(args []string) error {
	batch, err := parseAuthenticationSessionPruneArgs(args)
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
	application, err := wiring.InitApplicationWithConfig(cfg)
	if err != nil {
		return fmt.Errorf("initialize authentication session cleanup: %w", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	count, err := runAuthenticationSessionPrune(ctx, application.AuthenticationSessions, batch)
	if err != nil {
		return err
	}
	c.output.Success("Pruned %d authentication session(s)", count)
	return nil
}

func runAuthenticationSessionPrune(
	ctx context.Context,
	maintainer domain.AuthenticationSessionMaintainer,
	batch int,
) (int64, error) {
	if maintainer == nil {
		return 0, errAuthenticationSessionMaintainerUnavailable
	}
	return maintainer.PruneAuthenticationSessions(ctx, batch)
}

func parseAuthenticationSessionPruneArgs(args []string) (int, error) {
	batch := defaultAuthenticationSessionPruneBatch
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
			return 0, fmt.Errorf("unknown authentication session prune argument %q", argument)
		}
	}
	if batch < 1 || batch > maxAuthenticationSessionPruneBatch {
		return 0, fmt.Errorf("--batch must be between 1 and %d", maxAuthenticationSessionPruneBatch)
	}
	return batch, nil
}
