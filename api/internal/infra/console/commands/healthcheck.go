package commands

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/zgiai/luas/api/internal/infra/config"
)

const defaultHealthCheckTimeout = 2 * time.Second

// HealthCheckCommand probes the local liveness endpoint for container runtimes.
// It deliberately avoids loading application config so it can report health even
// when a dependency or production setting prevents the main process from starting.
type HealthCheckCommand struct {
	client *http.Client
	getenv func(string) string
}

// NewHealthCheckCommand creates the container-oriented liveness command.
func NewHealthCheckCommand() *HealthCheckCommand {
	return &HealthCheckCommand{
		client: &http.Client{
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		getenv: os.Getenv,
	}
}

func (c *HealthCheckCommand) Name() string { return "health:check" }
func (c *HealthCheckCommand) Description() string {
	return "Probe the local HTTP liveness endpoint"
}
func (c *HealthCheckCommand) Usage() string {
	return "health:check [--url=http://127.0.0.1:8025/health/live] [--timeout=2s]"
}

func (c *HealthCheckCommand) Run(args []string) error {
	target, timeout, err := c.options(args)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return fmt.Errorf("create health check request: %w", err)
	}

	client := c.client
	if client == nil {
		client = NewHealthCheckCommand().client
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("health check request failed: %w", err)
	}
	defer response.Body.Close()
	if _, err := io.Copy(io.Discard, io.LimitReader(response.Body, 4*1024)); err != nil {
		return fmt.Errorf("read health check response: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned HTTP %d", response.StatusCode)
	}
	return nil
}

func (c *HealthCheckCommand) options(args []string) (string, time.Duration, error) {
	getenv := c.getenv
	if getenv == nil {
		getenv = os.Getenv
	}

	target := strings.TrimSpace(getenv("HEALTHCHECK_URL"))
	timeout := defaultHealthCheckTimeout
	var err error

	for i := 0; i < len(args); i++ {
		argument := args[i]
		switch {
		case argument == "--url":
			i++
			if i >= len(args) {
				return "", 0, fmt.Errorf("--url requires a value")
			}
			target = args[i]
		case strings.HasPrefix(argument, "--url="):
			target = strings.TrimPrefix(argument, "--url=")
		case argument == "--timeout":
			i++
			if i >= len(args) {
				return "", 0, fmt.Errorf("--timeout requires a value")
			}
			timeout, err = time.ParseDuration(args[i])
			if err != nil {
				return "", 0, fmt.Errorf("invalid --timeout: %w", err)
			}
		case strings.HasPrefix(argument, "--timeout="):
			timeout, err = time.ParseDuration(strings.TrimPrefix(argument, "--timeout="))
			if err != nil {
				return "", 0, fmt.Errorf("invalid --timeout: %w", err)
			}
		default:
			return "", 0, fmt.Errorf("unknown option %q", argument)
		}
	}
	if target == "" {
		target, err = defaultHealthCheckURL(getenv)
		if err != nil {
			return "", 0, err
		}
	}

	if timeout <= 0 {
		return "", 0, fmt.Errorf("--timeout must be greater than zero")
	}
	parsed, err := url.ParseRequestURI(strings.TrimSpace(target))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", 0, fmt.Errorf("health check URL must be an absolute http or https URL")
	}

	return parsed.String(), timeout, nil
}

func defaultHealthCheckURL(getenv func(string) string) (string, error) {
	port := config.DefaultServerPort
	if rawPort := strings.TrimSpace(getenv("SERVER_PORT")); rawPort != "" {
		parsed, err := strconv.Atoi(rawPort)
		if err != nil || parsed < 1 || parsed > 65535 {
			return "", fmt.Errorf("SERVER_PORT must be an integer between 1 and 65535")
		}
		port = parsed
	}
	return fmt.Sprintf("http://127.0.0.1:%d/health/live", port), nil
}
