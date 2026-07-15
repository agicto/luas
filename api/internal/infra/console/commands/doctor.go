package commands

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/zgiai/luas/api/internal/capabilities/ai"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/infra/console"
)

// DoctorCommand validates the local configuration schema and the same typed
// configuration snapshot used by the runtime. It is intentionally read-only.
type DoctorCommand struct {
	output *console.Output
}

type doctorCheckLevel string

const (
	checkOK      doctorCheckLevel = "ok"
	checkWarning doctorCheckLevel = "warning"
	checkFailure doctorCheckLevel = "failure"
)

type doctorCheck struct {
	level  doctorCheckLevel
	label  string
	detail string
}

type doctorReport struct {
	checks []doctorCheck
}

type doctorConfigLoader func() (*config.Config, error)

func NewDoctorCommand() *DoctorCommand {
	return &DoctorCommand{output: console.NewOutput()}
}

func (c *DoctorCommand) Name() string { return "doctor" }
func (c *DoctorCommand) Description() string {
	return "Validate the environment schema and runtime configuration"
}
func (c *DoctorCommand) Usage() string { return "doctor [--env-example=path]" }

func (c *DoctorCommand) Run(args []string) error {
	envExample, err := parseDoctorArgs(args)
	if err != nil {
		return err
	}

	c.output.Title("luas doctor")
	report := auditDoctor(envExample, config.Load)
	for _, check := range report.checks {
		switch check.level {
		case checkOK:
			c.output.Line("  ✓ %s", check.label)
		case checkWarning:
			c.output.Warning("  ! %s — %s", check.label, check.detail)
		case checkFailure:
			c.output.Error("  ✗ %s — %s", check.label, check.detail)
		}
	}

	pass, warn, fail := report.counts()
	c.output.NewLine()
	c.output.Line("Result: %d ok, %d warning, %d fail", pass, warn, fail)
	if fail > 0 {
		return fmt.Errorf("doctor found %d failing check(s)", fail)
	}
	return nil
}

func parseDoctorArgs(args []string) (string, error) {
	flags := flag.NewFlagSet("doctor", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	envExample := flags.String("env-example", "", "path to the environment example schema")
	if err := flags.Parse(args); err != nil {
		return "", err
	}
	if flags.NArg() != 0 {
		return "", fmt.Errorf("doctor does not accept positional arguments")
	}
	if strings.TrimSpace(*envExample) != "" {
		return *envExample, nil
	}

	for _, candidate := range []string{".env.example", "api/.env.example"} {
		info, err := os.Stat(candidate)
		if err == nil && info.Mode().IsRegular() {
			return candidate, nil
		}
	}
	return ".env.example", nil
}

func auditDoctor(envExamplePath string, loadConfig doctorConfigLoader) doctorReport {
	report := doctorReport{}

	keys, err := readEnvKeys(envExamplePath)
	if err != nil {
		report.add(checkFailure, ".env.example schema", err.Error())
	} else {
		report.add(checkOK, fmt.Sprintf(".env.example schema (%d keys)", len(keys)), "")
	}

	cfg, err := loadConfig()
	if err != nil {
		report.add(checkFailure, "config.Load()", err.Error())
		return report
	}
	report.add(checkOK, "config.Load() succeeds", "")
	report.addConfigChecks(cfg)
	return report
}

func (r *doctorReport) addConfigChecks(cfg *config.Config) {
	if cfg.IsProduction() {
		r.add(checkOK, fmt.Sprintf("APP_ENV=%s (production defaults active)", cfg.App.Env), "")
	} else {
		r.add(checkOK, fmt.Sprintf("APP_ENV=%s (development defaults active)", cfg.App.Env), "")
	}

	r.add(
		checkOK,
		"opaque authentication sessions enabled",
		fmt.Sprintf(
			"absolute=%s idle=%s touch=%s retention=%s",
			cfg.Authentication.SessionTTL,
			cfg.Authentication.SessionIdleTimeout,
			cfg.Authentication.SessionTouchInterval,
			cfg.Authentication.SessionRetention,
		),
	)

	if hasWildcardOrigin(cfg.CORS.AllowOrigins) && cfg.CORS.AllowCredentials {
		r.add(checkFailure, "CORS '*' + credentials combination", "browsers reject this; set explicit origins")
	} else {
		r.add(checkOK, "CORS origins look sane", "")
	}

	if !cfg.AI.Enabled {
		r.add(checkOK, "AI capability is disabled", "")
		return
	}

	model := strings.TrimSpace(cfg.AI.DefaultModel)
	if model == "" {
		r.add(checkWarning, "AI enabled but AI_DEFAULT_MODEL is empty", "set a model id supported by the selected provider")
	} else {
		r.add(checkOK, fmt.Sprintf("AI_DEFAULT_MODEL=%s", model), "")
	}

	provider := strings.ToLower(strings.TrimSpace(cfg.AI.DefaultProvider))
	switch provider {
	case "":
		r.add(checkWarning, "AI enabled but AI_DEFAULT_PROVIDER is empty", "select a registered provider")
	case ai.ProviderOpenAI:
		if strings.TrimSpace(cfg.AI.OpenAI.APIKey) == "" {
			r.add(checkWarning, "AI enabled but OPENAI_API_KEY is empty", "ai:chat will fail for the built-in OpenAI provider")
		} else {
			r.add(checkOK, "OPENAI_API_KEY is set", "")
		}
	default:
		r.add(
			checkWarning,
			fmt.Sprintf("AI_DEFAULT_PROVIDER=%s requires a downstream adapter", provider),
			"implement the provider contract and register it in ai.NewManager",
		)
	}
}

func (r *doctorReport) add(level doctorCheckLevel, label, detail string) {
	r.checks = append(r.checks, doctorCheck{level: level, label: label, detail: detail})
}

func (r doctorReport) counts() (pass, warn, fail int) {
	for _, check := range r.checks {
		switch check.level {
		case checkOK:
			pass++
		case checkWarning:
			warn++
		case checkFailure:
			fail++
		}
	}
	return pass, warn, fail
}

func (r doctorReport) failures() int {
	_, _, fail := r.counts()
	return fail
}

func (r doctorReport) has(level doctorCheckLevel, text string) bool {
	for _, check := range r.checks {
		if check.level == level && strings.Contains(check.label+" "+check.detail, text) {
			return true
		}
	}
	return false
}

var envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// readEnvKeys validates and returns the names declared by an environment
// example. The file is a schema of available keys, not a list of values every
// deployment must set.
func readEnvKeys(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	keys := make([]string, 0)
	seen := make(map[string]int)
	scanner := bufio.NewScanner(f)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			return nil, fmt.Errorf("%s: line %d: expected KEY=VALUE", path, lineNumber)
		}
		key := strings.TrimSpace(line[:eq])
		if !envKeyPattern.MatchString(key) {
			return nil, fmt.Errorf("%s: line %d: invalid environment key %q", path, lineNumber, key)
		}
		if firstLine, duplicate := seen[key]; duplicate {
			return nil, fmt.Errorf("%s: line %d: duplicate key %s (first declared at line %d)", path, lineNumber, key, firstLine)
		}
		seen[key] = lineNumber
		keys = append(keys, key)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return keys, nil
}

func hasWildcardOrigin(origins []string) bool {
	for _, origin := range origins {
		if strings.TrimSpace(origin) == "*" {
			return true
		}
	}
	return false
}
