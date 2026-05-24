package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zgiai/zgo/internal/capabilities/ai"
	"github.com/zgiai/zgo/internal/infra/console"
	"github.com/zgiai/zgo/pkg/env"
)

// AIChatCommand sends a prompt to the configured AI provider.
type AIChatCommand struct {
	output *console.Output
}

func NewAIChatCommand() *AIChatCommand {
	return &AIChatCommand{output: console.NewOutput()}
}

func (c *AIChatCommand) Name() string        { return "ai:chat" }
func (c *AIChatCommand) Description() string { return "Send a prompt to the configured AI provider" }
func (c *AIChatCommand) Usage() string {
	return `ai:chat [--provider=openai] [--model=gpt-5.4] [--system="You are terse"] [--effort=low] "prompt"`
}

func (c *AIChatCommand) Run(args []string) error {
	req, err := parseAIChatArgs(args)
	if err != nil {
		return err
	}

	manager := ai.NewManager(loadAIConfig())
	resp, err := manager.GenerateText(context.Background(), req)
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

func loadAIConfig() ai.Config {
	return ai.Config{
		Enabled:         env.GetBool("AI_ENABLED", true),
		DefaultProvider: env.Get("AI_DEFAULT_PROVIDER", "openai"),
		DefaultModel:    env.Get("AI_DEFAULT_MODEL", "gpt-5.4"),
		RequestTimeout:  env.GetDuration("AI_REQUEST_TIMEOUT", 120*time.Second),
		OpenAI: ai.ProviderConfig{
			APIKey:  env.Get("OPENAI_API_KEY", ""),
			BaseURL: env.Get("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		},
		Anthropic: ai.ProviderConfig{
			APIKey:  env.Get("ANTHROPIC_API_KEY", ""),
			BaseURL: env.Get("ANTHROPIC_BASE_URL", ""),
		},
		Gemini: ai.ProviderConfig{
			APIKey:  env.Get("GEMINI_API_KEY", ""),
			BaseURL: env.Get("GEMINI_BASE_URL", ""),
		},
	}
}

func parseAIChatArgs(args []string) (*ai.TextRequest, error) {
	req := &ai.TextRequest{}
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
		case strings.HasPrefix(arg, "--"):
			return nil, fmt.Errorf("unknown flag: %s", arg)
		default:
			promptParts = append(promptParts, arg)
		}
	}

	req.Input = strings.TrimSpace(strings.Join(promptParts, " "))
	if req.Input == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	return req, nil
}
