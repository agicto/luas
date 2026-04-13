package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/zgiai/zgo/internal/infra/console"
	"github.com/zgiai/zgo/internal/infra/deploycontrol"
)

type DeployTargetsCommand struct {
	output *console.Output
}

func NewDeployTargetsCommand() *DeployTargetsCommand {
	return &DeployTargetsCommand{output: console.NewOutput()}
}

func (c *DeployTargetsCommand) Name() string        { return "deploy:targets" }
func (c *DeployTargetsCommand) Description() string { return "List configured deployment targets" }
func (c *DeployTargetsCommand) Usage() string       { return "deploy:targets" }

func (c *DeployTargetsCommand) Run(args []string) error {
	targets, err := deploycontrol.NewManager().ListTargets()
	if err != nil {
		return err
	}

	c.output.Title("Deployment Targets")
	if len(targets) == 0 {
		c.output.Warning("No deployment targets configured. Create deploy.targets.json first.")
		return nil
	}

	rows := make([][]string, 0, len(targets))
	for _, target := range targets {
		rows = append(rows, []string{
			target.Name,
			firstNonEmpty(target.DisplayName, target.Name),
			firstNonEmpty(target.Provider, "shell"),
			target.WorkingDirectory,
			string(firstCertificateMode(target)),
		})
	}

	c.output.Table([]string{"Name", "Display", "Provider", "Directory", "TLS"}, rows)
	return nil
}

type DeployListCommand struct {
	output *console.Output
}

func NewDeployListCommand() *DeployListCommand {
	return &DeployListCommand{output: console.NewOutput()}
}

func (c *DeployListCommand) Name() string        { return "deploy:list" }
func (c *DeployListCommand) Description() string { return "List recent deployments" }
func (c *DeployListCommand) Usage() string       { return "deploy:list [--limit=20]" }

func (c *DeployListCommand) Run(args []string) error {
	limit := 20
	for _, arg := range args {
		if strings.HasPrefix(arg, "--limit=") {
			fmt.Sscanf(strings.TrimPrefix(arg, "--limit="), "%d", &limit)
		}
	}

	deployments, err := deploycontrol.NewManager().ListDeployments(limit)
	if err != nil {
		return err
	}

	c.output.Title("Recent Deployments")
	if len(deployments) == 0 {
		c.output.Warning("No deployments found.")
		return nil
	}

	rows := make([][]string, 0, len(deployments))
	for _, deployment := range deployments {
		rows = append(rows, []string{
			deployment.ID,
			deployment.Target,
			string(deployment.Status),
			deployment.TriggeredBy,
			deployment.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.output.Table([]string{"ID", "Target", "Status", "By", "Created"}, rows)
	return nil
}

type DeployRunCommand struct {
	output *console.Output
}

func NewDeployRunCommand() *DeployRunCommand {
	return &DeployRunCommand{output: console.NewOutput()}
}

func (c *DeployRunCommand) Name() string        { return "deploy:run" }
func (c *DeployRunCommand) Description() string { return "Run a deployment from the CLI" }
func (c *DeployRunCommand) Usage() string {
	return "deploy:run --target=<name> [--branch=main] [--commit=sha] [--by=cli] [--env=KEY=value]"
}

func (c *DeployRunCommand) Run(args []string) error {
	req, err := parseDeployRunArgs(args)
	if err != nil {
		return err
	}

	manager := deploycontrol.NewManager()

	c.output.Title("Deployment Run")
	var lastStatus string
	deployment, err := manager.RunDeployment(context.Background(), req, func(event deploycontrol.WatchEvent) {
		if event.Log != nil {
			prefix := "info"
			if event.Log.Stream == "stderr" {
				prefix = "err "
			}
			c.output.Line("[%s] %s", prefix, event.Log.Message)
		}
		if event.Deployment != nil {
			lastStatus = string(event.Deployment.Status)
		}
	})
	if err != nil {
		if deployment != nil {
			c.output.Error("Deployment %s failed", deployment.ID)
		}
		return err
	}

	c.output.Success("Deployment %s finished with status %s", deployment.ID, firstNonEmpty(lastStatus, string(deployment.Status)))
	return nil
}

type DeployLogsCommand struct {
	output *console.Output
}

func NewDeployLogsCommand() *DeployLogsCommand {
	return &DeployLogsCommand{output: console.NewOutput()}
}

func (c *DeployLogsCommand) Name() string        { return "deploy:logs" }
func (c *DeployLogsCommand) Description() string { return "Show deployment logs from local storage" }
func (c *DeployLogsCommand) Usage() string       { return "deploy:logs <deployment-id> [--tail=200]" }

func (c *DeployLogsCommand) Run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("deployment id is required")
	}

	deploymentID := args[0]
	tail := 200
	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "--tail=") {
			fmt.Sscanf(strings.TrimPrefix(arg, "--tail="), "%d", &tail)
		}
	}

	logs, err := deploycontrol.NewManager().ListLogs(deploymentID, tail)
	if err != nil {
		return err
	}

	for _, entry := range logs {
		c.output.Line("[%s] %s", entry.Stream, entry.Message)
	}
	return nil
}

type DeployCertificateCommand struct {
	output *console.Output
}

func NewDeployCertificateCommand() *DeployCertificateCommand {
	return &DeployCertificateCommand{output: console.NewOutput()}
}

func (c *DeployCertificateCommand) Name() string { return "deploy:cert" }
func (c *DeployCertificateCommand) Description() string {
	return "Generate a self-signed TLS certificate"
}
func (c *DeployCertificateCommand) Usage() string {
	return "deploy:cert --domain=example.com [--days=90]"
}

func (c *DeployCertificateCommand) Run(args []string) error {
	req := deploycontrol.CertificateRequest{ValidDays: 90}

	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--domain="):
			req.Domain = strings.TrimSpace(strings.TrimPrefix(arg, "--domain="))
		case strings.HasPrefix(arg, "--days="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--days="), "%d", &req.ValidDays)
		}
	}

	if req.Domain == "" {
		return fmt.Errorf("domain is required")
	}

	cert, err := deploycontrol.NewManager().GenerateCertificate(req)
	if err != nil {
		return err
	}

	c.output.Title("Certificate Generated")
	c.output.TwoColumn("Domain", cert.Domain)
	c.output.TwoColumn("Certificate", cert.CertPath)
	c.output.TwoColumn("Private Key", cert.KeyPath)
	c.output.TwoColumn("Expires", cert.ExpiresAt.Format("2006-01-02 15:04:05"))
	return nil
}

func parseDeployRunArgs(args []string) (deploycontrol.RunRequest, error) {
	req := deploycontrol.RunRequest{
		TriggeredBy: "cli",
		TriggerMode: "cli",
		Environment: map[string]string{},
	}

	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--target="):
			req.Target = strings.TrimSpace(strings.TrimPrefix(arg, "--target="))
		case strings.HasPrefix(arg, "--branch="):
			req.Branch = strings.TrimSpace(strings.TrimPrefix(arg, "--branch="))
		case strings.HasPrefix(arg, "--commit="):
			req.Commit = strings.TrimSpace(strings.TrimPrefix(arg, "--commit="))
		case strings.HasPrefix(arg, "--by="):
			req.TriggeredBy = strings.TrimSpace(strings.TrimPrefix(arg, "--by="))
		case strings.HasPrefix(arg, "--env="):
			pair := strings.TrimSpace(strings.TrimPrefix(arg, "--env="))
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) != 2 {
				return req, fmt.Errorf("invalid --env value: %s", pair)
			}
			req.Environment[parts[0]] = parts[1]
		default:
			return req, fmt.Errorf("unknown flag: %s", arg)
		}
	}

	if req.Target == "" {
		return req, fmt.Errorf("target is required")
	}
	if len(req.Environment) == 0 {
		req.Environment = nil
	}
	return req, nil
}

func firstCertificateMode(target deploycontrol.TargetConfig) deploycontrol.CertificateMode {
	if target.CertificateMode != "" {
		return target.CertificateMode
	}
	if strings.EqualFold(target.Provider, "render") && target.Domain != "" {
		return deploycontrol.CertificateModeRenderManaged
	}
	return deploycontrol.CertificateModeDisabled
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
