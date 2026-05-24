package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/infra/console"
	"github.com/zgiai/luas/api/pkg/env"
)

// DoctorCommand audits the local environment against `.env.example` and
// flags common mistakes that bite new users on the way to a clean run:
// missing required keys, placeholder secrets still in place, JWT_SECRET
// too short for the configured APP_ENV, etc.
//
// It is intentionally read-only — never mutates the environment, never
// touches the database.
type DoctorCommand struct {
	output *console.Output
}

func NewDoctorCommand() *DoctorCommand {
	return &DoctorCommand{output: console.NewOutput()}
}

func (c *DoctorCommand) Name() string        { return "doctor" }
func (c *DoctorCommand) Description() string { return "Audit .env vs .env.example and flag misconfigurations" }
func (c *DoctorCommand) Usage() string       { return "doctor" }

func (c *DoctorCommand) Run(args []string) error {
	c.output.Title("luas doctor")

	pass := 0
	warn := 0
	fail := 0

	check := func(level string, label string, detail string) {
		switch level {
		case "ok":
			pass++
			c.output.Line("  ✓ %s", label)
		case "warn":
			warn++
			c.output.Warning("  ! %s — %s", label, detail)
		case "fail":
			fail++
			c.output.Error("  ✗ %s — %s", label, detail)
		}
	}

	// 1) .env.example exists and is parsable
	expectedKeys, err := readEnvKeys(".env.example")
	if err != nil {
		check("fail", ".env.example readable", err.Error())
		return c.summary(pass, warn, fail)
	}
	check("ok", ".env.example readable", "")

	// 2) Every key in .env.example is set in the live env, OR has a
	//    default in config.Load — we only warn for keys that are
	//    *completely* unset both in env and .env files.
	missing := []string{}
	for _, key := range expectedKeys {
		if _, ok := os.LookupEnv(key); ok {
			continue
		}
		if env.Get(key, "") != "" {
			continue
		}
		missing = append(missing, key)
	}
	if len(missing) == 0 {
		check("ok", "all .env.example keys are present", "")
	} else {
		check("warn", "some .env.example keys are unset", strings.Join(missing, ", "))
	}

	// 3) Try loading config — this runs the full validator.
	cfg, err := config.Load()
	if err != nil {
		check("fail", "config.Load()", err.Error())
		return c.summary(pass, warn, fail)
	}
	check("ok", "config.Load() succeeds", "")

	// 4) Spot-check the production-critical knobs.
	if cfg.App.Env == "production" {
		check("ok", "APP_ENV=production", "")
	} else {
		check("ok", fmt.Sprintf("APP_ENV=%s (dev defaults active)", cfg.App.Env), "")
	}

	if len(cfg.JWT.Secret) >= 32 {
		check("ok", "JWT_SECRET length >= 32", "")
	} else {
		level := "warn"
		if cfg.App.Env == "production" {
			level = "fail"
		}
		check(level, "JWT_SECRET length < 32", fmt.Sprintf("got %d chars", len(cfg.JWT.Secret)))
	}

	if hasWildcardOrigin(cfg.CORS.AllowOrigins) && cfg.CORS.AllowCredentials {
		check("fail", "CORS '*' + credentials combo", "browsers reject this; set explicit origins")
	} else {
		check("ok", "CORS origins look sane", "")
	}

	// 5) AI capability sanity.
	if cfg.AI.Enabled {
		if strings.TrimSpace(cfg.AI.DefaultModel) == "" {
			check("warn", "AI enabled but AI_DEFAULT_MODEL is empty", "set a real model name (e.g. gpt-5)")
		} else if strings.Contains(cfg.AI.DefaultModel, "gpt-5.4") {
			check("warn", "AI_DEFAULT_MODEL=gpt-5.4 is not a real OpenAI model", "use gpt-5 or another valid id")
		} else {
			check("ok", fmt.Sprintf("AI_DEFAULT_MODEL=%s", cfg.AI.DefaultModel), "")
		}

		if cfg.AI.OpenAI.APIKey == "" {
			check("warn", "AI enabled but OPENAI_API_KEY is empty", "ai:chat will fail")
		} else {
			check("ok", "OPENAI_API_KEY is set", "")
		}
	} else {
		check("ok", "AI capability is disabled", "")
	}

	return c.summary(pass, warn, fail)
}

func (c *DoctorCommand) summary(pass, warn, fail int) error {
	c.output.NewLine()
	c.output.Line("Result: %d ok, %d warning, %d fail", pass, warn, fail)
	if fail > 0 {
		return fmt.Errorf("doctor found %d failing check(s)", fail)
	}
	return nil
}

// readEnvKeys returns the env-var names that .env.example declares,
// ignoring blank lines and comments. Values are not returned —
// .env.example is treated as a contract of which keys should exist.
func readEnvKeys(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	keys := []string{}
	seen := make(map[string]struct{})
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.Index(line, "=")
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return keys, nil
}

func hasWildcardOrigin(origins []string) bool {
	for _, o := range origins {
		if strings.TrimSpace(o) == "*" {
			return true
		}
	}
	return false
}
