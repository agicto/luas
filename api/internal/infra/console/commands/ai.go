package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/zgiai/luas/api/internal/capabilities/ai"
	"github.com/zgiai/luas/api/internal/infra/console"
	"github.com/zgiai/luas/api/pkg/env"
)

// AIChatCommand sends a prompt to the configured AI provider.
//
// By default it streams the response token-by-token (when the provider
// supports it). Pass --no-stream for a one-shot response.
type AIChatCommand struct {
	output *console.Output
}

func NewAIChatCommand() *AIChatCommand {
	return &AIChatCommand{output: console.NewOutput()}
}

func (c *AIChatCommand) Name() string        { return "ai:chat" }
func (c *AIChatCommand) Description() string { return "Send a prompt to the configured AI provider" }
func (c *AIChatCommand) Usage() string {
	return `ai:chat [--provider=openai] [--model=gpt-5] [--system="You are terse"] [--effort=low] [--no-stream] "prompt"`
}

func (c *AIChatCommand) Run(args []string) error {
	req, noStream, err := parseAIChatArgs(args)
	if err != nil {
		return err
	}

	manager := ai.NewManager(loadAIConfig())
	ctx := context.Background()

	if noStream {
		return c.runOneShot(ctx, manager, req)
	}

	// Try streaming first; fall back to one-shot if the provider doesn't
	// support it. This keeps the CLI working with any future provider that
	// only implements the base Provider interface.
	if err := c.runStream(ctx, manager, req); err != nil {
		if errors.Is(err, ai.ErrStreamingUnsupported) {
			return c.runOneShot(ctx, manager, req)
		}
		return err
	}
	return nil
}

func (c *AIChatCommand) runOneShot(ctx context.Context, m *ai.Manager, req *ai.TextRequest) error {
	resp, err := m.GenerateText(ctx, req)
	if err != nil {
		return err
	}
	c.output.Title("AI Response")
	c.output.TwoColumn("Provider", resp.Provider)
	c.output.TwoColumn("Model", resp.Model)
	c.output.NewLine()
	c.output.Line("%s", resp.Text)
	return nil
}

func (c *AIChatCommand) runStream(ctx context.Context, m *ai.Manager, req *ai.TextRequest) error {
	ch, err := m.GenerateTextStream(ctx, req)
	if err != nil {
		return err
	}
	for chunk := range ch {
		if chunk.Err != nil {
			fmt.Fprintln(os.Stdout) // newline after partial output
			return chunk.Err
		}
		// Write directly to stdout so terminals flush per chunk.
		fmt.Fprint(os.Stdout, chunk.Delta)
	}
	fmt.Fprintln(os.Stdout)
	return nil
}

func loadAIConfig() ai.Config {
	return ai.Config{
		Enabled:         env.GetBool("AI_ENABLED", true),
		DefaultProvider: env.Get("AI_DEFAULT_PROVIDER", "openai"),
		DefaultModel:    env.Get("AI_DEFAULT_MODEL", "gpt-5"),
		RequestTimeout:  env.GetDuration("AI_REQUEST_TIMEOUT", 120*time.Second),
		OpenAI: ai.ProviderConfig{
			APIKey:  env.Get("OPENAI_API_KEY", ""),
			BaseURL: env.Get("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		},
	}
}

func parseAIChatArgs(args []string) (*ai.TextRequest, bool, error) {
	req := &ai.TextRequest{}
	noStream := false
	promptParts := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case arg == "--provider" && i+1 < len(args):
			i++
			req.Provider = args[i]
		case strings.HasPrefix(arg, "--provider="):
			req.Provider = strings.TrimPrefix(arg, "--provider=")
		case arg == "--model" && i+1 < len(args):
			i++
			req.Model = args[i]
		case strings.HasPrefix(arg, "--model="):
			req.Model = strings.TrimPrefix(arg, "--model=")
		case (arg == "--system" || arg == "--instructions") && i+1 < len(args):
			i++
			req.Instructions = args[i]
		case strings.HasPrefix(arg, "--system="):
			req.Instructions = strings.TrimPrefix(arg, "--system=")
		case strings.HasPrefix(arg, "--instructions="):
			req.Instructions = strings.TrimPrefix(arg, "--instructions=")
		case arg == "--effort" && i+1 < len(args):
			i++
			req.ReasoningEffort = args[i]
		case strings.HasPrefix(arg, "--effort="):
			req.ReasoningEffort = strings.TrimPrefix(arg, "--effort=")
		case arg == "--no-stream":
			noStream = true
		case strings.HasPrefix(arg, "--"):
			return nil, false, fmt.Errorf("unknown flag: %s", arg)
		default:
			promptParts = append(promptParts, arg)
		}
	}

	req.Input = strings.TrimSpace(strings.Join(promptParts, " "))
	if req.Input == "" {
		return nil, false, fmt.Errorf("prompt is required")
	}

	return req, noStream, nil
}
