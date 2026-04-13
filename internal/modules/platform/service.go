package platform

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	cryptocap "github.com/zgiai/zgo/internal/capabilities/crypto"
	"github.com/zgiai/zgo/internal/infra/deploycontrol"
	"github.com/zgiai/zgo/pkg/env"
)

var slugPattern = regexp.MustCompile(`[^a-z0-9]+`)

type PlatformService interface {
	Overview(ctx context.Context) (*PlatformOverview, error)
	ListDeployTargets() ([]deploycontrol.TargetConfig, error)
	ListGitHubConnections(ctx context.Context) ([]GitHubConnection, error)
	ConnectGitHub(ctx context.Context, req *ConnectGitHubRequest) (*GitHubConnection, error)
	ListGitHubRepositories(ctx context.Context, connectionID uint, query string, limit int) ([]GitHubRepository, error)
	ListProjects(ctx context.Context) ([]Project, error)
	CreateProject(ctx context.Context, req *CreateProjectRequest) (*Project, error)
	ListServices(ctx context.Context) ([]Service, error)
	GetService(ctx context.Context, id uint) (*Service, error)
	ImportService(ctx context.Context, req *ImportServiceRequest) (*Service, error)
	UpdateService(ctx context.Context, id uint, req *UpdateServiceRequest) (*Service, error)
	ReplaceEnvironment(ctx context.Context, serviceID uint, req *ReplaceEnvironmentRequest) ([]ServiceEnvironmentVariable, error)
	ListServiceDeployments(ctx context.Context, serviceID uint, limit int) ([]ServiceDeploymentSnapshot, error)
	DeployService(ctx context.Context, serviceID uint, req *TriggerServiceDeploymentRequest) (*Service, *deploycontrol.Deployment, error)
	HandleGitHubWebhook(ctx context.Context, serviceID uint, signature string, payload []byte) (map[string]any, error)
	ListDeploymentLogs(id string, tail int) ([]deploycontrol.LogEntry, error)
	WatchDeployment(id string) (<-chan deploycontrol.WatchEvent, func(), error)
}

type service struct {
	repo          *repository
	manager       *deploycontrol.Manager
	github        GitHubClient
	storageRoot   string
	workspaceRoot string
}

func NewService(repo *repository, manager *deploycontrol.Manager, github GitHubClient) PlatformService {
	storageRoot := env.Get("PLATFORM_STORAGE_ROOT", filepath.Join("storage", "platform"))
	if absoluteRoot, err := filepath.Abs(storageRoot); err == nil {
		storageRoot = absoluteRoot
	}
	workspaceRoot := filepath.Join(storageRoot, "releases")
	_ = os.MkdirAll(workspaceRoot, 0o755)

	return &service{
		repo:          repo,
		manager:       manager,
		github:        github,
		storageRoot:   storageRoot,
		workspaceRoot: workspaceRoot,
	}
}

func (s *service) Overview(ctx context.Context) (*PlatformOverview, error) {
	projectCount, err := s.repo.CountProjects(ctx)
	if err != nil {
		return nil, err
	}
	serviceCount, err := s.repo.CountServices(ctx)
	if err != nil {
		return nil, err
	}
	connectionCount, err := s.repo.CountGitHubConnections(ctx)
	if err != nil {
		return nil, err
	}

	projects, err := s.ListProjects(ctx)
	if err != nil {
		return nil, err
	}
	services, err := s.ListServices(ctx)
	if err != nil {
		return nil, err
	}

	recentDeployments, err := s.manager.ListDeployments(12)
	if err != nil {
		return nil, err
	}

	deploymentSnapshots := make([]ServiceDeploymentSnapshot, 0, len(recentDeployments))
	for _, item := range recentDeployments {
		deploymentSnapshots = append(deploymentSnapshots, snapshotFromDeployment(item))
	}

	return &PlatformOverview{
		Projects:          int(projectCount),
		Services:          int(serviceCount),
		GitHubConnections: int(connectionCount),
		RecentDeployments: len(deploymentSnapshots),
		ProjectsList:      projects,
		ServicesList:      services,
		Deployments:       deploymentSnapshots,
	}, nil
}

func (s *service) ListDeployTargets() ([]deploycontrol.TargetConfig, error) {
	return s.manager.ListTargets()
}

func (s *service) ListGitHubConnections(ctx context.Context) ([]GitHubConnection, error) {
	rows, err := s.repo.ListGitHubConnections(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]GitHubConnection, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.toView())
	}
	return result, nil
}

func (s *service) ConnectGitHub(ctx context.Context, req *ConnectGitHubRequest) (*GitHubConnection, error) {
	token := strings.TrimSpace(req.Token)
	if token == "" {
		return nil, fmt.Errorf("github token is required")
	}

	viewer, err := s.github.GetViewer(ctx, token)
	if err != nil {
		return nil, err
	}

	encryptedToken, err := s.encryptToken(token)
	if err != nil {
		return nil, err
	}

	now := currentUTCPtr()
	row := &GitHubConnectionPO{
		Name:           firstNonEmpty(strings.TrimSpace(req.Name), viewer.DisplayName, viewer.Login),
		Provider:       "github",
		Login:          viewer.Login,
		DisplayName:    viewer.DisplayName,
		AvatarURL:      viewer.AvatarURL,
		TokenEncrypted: encryptedToken,
		TokenMasked:    maskToken(token),
		LastSyncedAt:   now,
	}

	if err := s.repo.UpsertGitHubConnection(ctx, row); err != nil {
		return nil, err
	}

	view := row.toView()
	return &view, nil
}

func (s *service) ListGitHubRepositories(ctx context.Context, connectionID uint, query string, limit int) ([]GitHubRepository, error) {
	connection, err := s.repo.GetGitHubConnection(ctx, connectionID)
	if err != nil {
		return nil, err
	}

	token, err := s.decryptToken(connection.TokenEncrypted)
	if err != nil {
		return nil, err
	}

	return s.github.ListRepositories(ctx, token, query, limit)
}

func (s *service) ListProjects(ctx context.Context) ([]Project, error) {
	projects, serviceCount, err := s.repo.ListProjects(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]Project, 0, len(projects))
	for _, project := range projects {
		result = append(result, project.toView(serviceCount[project.ID]))
	}
	return result, nil
}

func (s *service) CreateProject(ctx context.Context, req *CreateProjectRequest) (*Project, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("project name is required")
	}

	slug, err := s.nextProjectSlug(ctx, name)
	if err != nil {
		return nil, err
	}

	project := &ProjectPO{
		Name:             name,
		Slug:             slug,
		Description:      strings.TrimSpace(req.Description),
		ProductionDomain: strings.TrimSpace(req.ProductionDomain),
	}
	if err := s.repo.CreateProject(ctx, project); err != nil {
		return nil, err
	}

	view := project.toView(0)
	return &view, nil
}

func (s *service) ListServices(ctx context.Context) ([]Service, error) {
	rows, err := s.repo.ListServices(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]Service, 0, len(rows))
	for _, row := range rows {
		view, buildErr := s.buildServiceView(ctx, row)
		if buildErr != nil {
			return nil, buildErr
		}
		result = append(result, *view)
	}
	return result, nil
}

func (s *service) GetService(ctx context.Context, id uint) (*Service, error) {
	record, err := s.repo.GetServiceRecord(ctx, id, true)
	if err != nil {
		return nil, err
	}

	return s.buildServiceView(ctx, *record)
}

func (s *service) ImportService(ctx context.Context, req *ImportServiceRequest) (*Service, error) {
	recordProject, err := s.ensureProject(ctx, req)
	if err != nil {
		return nil, err
	}

	connection, err := s.repo.GetGitHubConnection(ctx, req.GitHubConnectionID)
	if err != nil {
		return nil, err
	}

	if _, err := s.manager.GetTarget(req.DeployTarget); err != nil {
		return nil, err
	}

	serviceName := strings.TrimSpace(req.Name)
	if serviceName == "" {
		serviceName = strings.TrimSpace(req.RepositoryName)
	}
	serviceSlug, err := s.nextServiceSlug(ctx, serviceName)
	if err != nil {
		return nil, err
	}

	branch := strings.TrimSpace(req.Branch)
	if branch == "" {
		return nil, fmt.Errorf("branch is required")
	}

	serviceRow := &ServicePO{
		ProjectID:          recordProject.ID,
		GitHubConnectionID: connection.ID,
		Name:               serviceName,
		Slug:               serviceSlug,
		RepositoryOwner:    strings.TrimSpace(req.RepositoryOwner),
		RepositoryName:     strings.TrimSpace(req.RepositoryName),
		RepositoryURL:      firstNonEmpty(strings.TrimSpace(req.RepositoryURL), fmt.Sprintf("https://github.com/%s/%s.git", strings.TrimSpace(req.RepositoryOwner), strings.TrimSpace(req.RepositoryName))),
		DefaultBranch:      branch,
		RootDirectory:      strings.TrimSpace(req.RootDirectory),
		DeployStrategy:     normalizeStrategy(req.DeployStrategy),
		DeployTarget:       strings.TrimSpace(req.DeployTarget),
		DockerfilePath:     strings.TrimSpace(req.DockerfilePath),
		ComposeFile:        strings.TrimSpace(req.ComposeFile),
		BuildCommand:       strings.TrimSpace(req.BuildCommand),
		DeployCommand:      strings.TrimSpace(req.DeployCommand),
		HealthCheckURL:     strings.TrimSpace(req.HealthCheckURL),
		Domain:             strings.TrimSpace(req.Domain),
		PublishedPort:      req.PublishedPort,
		ContainerPort:      req.ContainerPort,
		AutoDeployEnabled:  req.AutoDeployEnabled,
		WebhookSecret:      randomSecret(),
	}

	if err := validateServiceConfig(serviceRow); err != nil {
		return nil, err
	}

	if err := s.repo.CreateService(ctx, serviceRow); err != nil {
		return nil, err
	}

	envRows := buildEnvRows(serviceRow.ID, req.Environment)
	if err := s.repo.ReplaceServiceEnvVars(ctx, serviceRow.ID, envRows); err != nil {
		return nil, err
	}

	return s.GetService(ctx, serviceRow.ID)
}

func (s *service) UpdateService(ctx context.Context, id uint, req *UpdateServiceRequest) (*Service, error) {
	record, err := s.repo.MustGetServiceRecord(ctx, id)
	if err != nil {
		return nil, err
	}

	if value := strings.TrimSpace(req.Name); value != "" && value != record.Service.Name {
		record.Service.Name = value
		slug, slugErr := s.nextServiceSlug(ctx, value)
		if slugErr != nil {
			return nil, slugErr
		}
		record.Service.Slug = slug
	}
	if value := strings.TrimSpace(req.Branch); value != "" {
		record.Service.DefaultBranch = value
	}
	if value := strings.TrimSpace(req.RootDirectory); value != "" {
		record.Service.RootDirectory = value
	}
	if value := strings.TrimSpace(req.DeployStrategy); value != "" {
		record.Service.DeployStrategy = normalizeStrategy(value)
	}
	if value := strings.TrimSpace(req.DeployTarget); value != "" {
		if _, targetErr := s.manager.GetTarget(value); targetErr != nil {
			return nil, targetErr
		}
		record.Service.DeployTarget = value
	}
	if value := strings.TrimSpace(req.DockerfilePath); value != "" {
		record.Service.DockerfilePath = value
	}
	if value := strings.TrimSpace(req.ComposeFile); value != "" {
		record.Service.ComposeFile = value
	}
	if value := strings.TrimSpace(req.BuildCommand); value != "" {
		record.Service.BuildCommand = value
	}
	if value := strings.TrimSpace(req.DeployCommand); value != "" {
		record.Service.DeployCommand = value
	}
	if value := strings.TrimSpace(req.HealthCheckURL); value != "" {
		record.Service.HealthCheckURL = value
	}
	if value := strings.TrimSpace(req.Domain); value != "" {
		record.Service.Domain = value
	}
	if req.PublishedPort > 0 {
		record.Service.PublishedPort = req.PublishedPort
	}
	if req.ContainerPort > 0 {
		record.Service.ContainerPort = req.ContainerPort
	}
	if req.AutoDeployEnabled != nil {
		record.Service.AutoDeployEnabled = *req.AutoDeployEnabled
	}

	if err := validateServiceConfig(&record.Service); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateService(ctx, &record.Service); err != nil {
		return nil, err
	}

	return s.GetService(ctx, id)
}

func (s *service) ReplaceEnvironment(ctx context.Context, serviceID uint, req *ReplaceEnvironmentRequest) ([]ServiceEnvironmentVariable, error) {
	if _, err := s.repo.MustGetServiceRecord(ctx, serviceID); err != nil {
		return nil, err
	}

	rows := buildEnvRows(serviceID, req.Variables)
	if err := s.repo.ReplaceServiceEnvVars(ctx, serviceID, rows); err != nil {
		return nil, err
	}

	updated, err := s.repo.ListServiceEnvVars(ctx, serviceID)
	if err != nil {
		return nil, err
	}

	result := make([]ServiceEnvironmentVariable, 0, len(updated))
	for _, row := range updated {
		result = append(result, row.toView())
	}
	return result, nil
}

func (s *service) ListServiceDeployments(ctx context.Context, serviceID uint, limit int) ([]ServiceDeploymentSnapshot, error) {
	if _, err := s.repo.MustGetServiceRecord(ctx, serviceID); err != nil {
		return nil, err
	}

	deployments, err := s.manager.ListDeploymentsByLabel("service_id", strconv.FormatUint(uint64(serviceID), 10), limit)
	if err != nil {
		return nil, err
	}

	result := make([]ServiceDeploymentSnapshot, 0, len(deployments))
	for _, deployment := range deployments {
		result = append(result, snapshotFromDeployment(deployment))
	}
	return result, nil
}

func (s *service) DeployService(ctx context.Context, serviceID uint, req *TriggerServiceDeploymentRequest) (*Service, *deploycontrol.Deployment, error) {
	record, err := s.repo.MustGetServiceRecord(ctx, serviceID)
	if err != nil {
		return nil, nil, err
	}

	connectionToken, err := s.decryptToken(record.Connection.TokenEncrypted)
	if err != nil {
		return nil, nil, err
	}

	branch := firstNonEmpty(strings.TrimSpace(req.Branch), record.Service.DefaultBranch)
	releaseRoot, sourceRoot, err := s.prepareReleaseWorkspace(ctx, record, connectionToken, branch, strings.TrimSpace(req.Commit))
	if err != nil {
		return nil, nil, err
	}

	workingDir, err := resolveWorkingDirectory(sourceRoot, record.Service.RootDirectory)
	if err != nil {
		return nil, nil, err
	}

	envFile, envMap, err := s.writeEnvironmentFile(workingDir, record)
	if err != nil {
		return nil, nil, err
	}

	target, err := s.manager.GetTarget(record.Service.DeployTarget)
	if err != nil {
		return nil, nil, err
	}

	plan, err := s.buildPlan(record, target, releaseRoot, workingDir, envFile, envMap, branch, strings.TrimSpace(req.Commit), strings.TrimSpace(req.TriggeredBy))
	if err != nil {
		return nil, nil, err
	}

	deployment, err := s.manager.StartPlan(plan)
	if err != nil {
		return nil, nil, err
	}

	record.Service.LastDeploymentID = deployment.ID
	record.Service.LastDeployError = ""
	if err := s.repo.UpdateService(ctx, &record.Service); err != nil {
		return nil, nil, err
	}

	view, err := s.buildServiceView(ctx, *record)
	if err != nil {
		return nil, nil, err
	}

	return view, deployment, nil
}

func (s *service) ListDeploymentLogs(id string, tail int) ([]deploycontrol.LogEntry, error) {
	return s.manager.ListLogs(id, tail)
}

func (s *service) HandleGitHubWebhook(ctx context.Context, serviceID uint, signature string, payload []byte) (map[string]any, error) {
	record, err := s.repo.MustGetServiceRecord(ctx, serviceID)
	if err != nil {
		return nil, err
	}

	if !record.Service.AutoDeployEnabled {
		return map[string]any{
			"status": "ignored",
			"reason": "auto deploy disabled",
		}, nil
	}

	if record.Service.WebhookSecret != "" && !verifyGitHubSignature(signature, payload, record.Service.WebhookSecret) {
		return nil, fmt.Errorf("invalid webhook signature")
	}

	var event struct {
		Ref   string `json:"ref"`
		After string `json:"after"`
	}
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}

	branch := strings.TrimPrefix(strings.TrimSpace(event.Ref), "refs/heads/")
	if branch == "" {
		return map[string]any{
			"status": "ignored",
			"reason": "missing branch ref",
		}, nil
	}
	if branch != record.Service.DefaultBranch {
		return map[string]any{
			"status": "ignored",
			"reason": "branch not configured for auto deploy",
			"branch": branch,
		}, nil
	}

	serviceView, deployment, err := s.DeployService(ctx, serviceID, &TriggerServiceDeploymentRequest{
		Branch:      branch,
		Commit:      strings.TrimSpace(event.After),
		TriggeredBy: "github-webhook",
	})
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"status":     "accepted",
		"service":    serviceView,
		"deployment": deployment,
	}, nil
}

func (s *service) WatchDeployment(id string) (<-chan deploycontrol.WatchEvent, func(), error) {
	return s.manager.Watch(id)
}

func (s *service) ensureProject(ctx context.Context, req *ImportServiceRequest) (*ProjectPO, error) {
	if req.ProjectID > 0 {
		return s.repo.GetProject(ctx, req.ProjectID)
	}

	name := strings.TrimSpace(req.ProjectName)
	if name == "" {
		name = strings.TrimSpace(req.RepositoryName)
	}
	project, err := s.CreateProject(ctx, &CreateProjectRequest{
		Name:             name,
		Description:      req.ProjectDescription,
		ProductionDomain: req.ProjectProductionDomain,
	})
	if err != nil {
		return nil, err
	}
	return s.repo.GetProject(ctx, project.ID)
}

func (s *service) nextProjectSlug(ctx context.Context, value string) (string, error) {
	return s.nextSlug(ctx, value, s.repo.ProjectSlugExists)
}

func (s *service) nextServiceSlug(ctx context.Context, value string) (string, error) {
	return s.nextSlug(ctx, value, s.repo.ServiceSlugExists)
}

func (s *service) nextSlug(ctx context.Context, value string, exists func(context.Context, string) (bool, error)) (string, error) {
	base := slugify(value)
	candidate := base
	index := 2
	for {
		inUse, err := exists(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !inUse {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, index)
		index++
	}
}

func (s *service) buildServiceView(ctx context.Context, record serviceRecord) (*Service, error) {
	view := &Service{
		ID:                 record.Service.ID,
		ProjectID:          record.Project.ID,
		ProjectName:        record.Project.Name,
		GitHubConnectionID: record.Connection.ID,
		GitHubLogin:        record.Connection.Login,
		Name:               record.Service.Name,
		Slug:               record.Service.Slug,
		RepositoryOwner:    record.Service.RepositoryOwner,
		RepositoryName:     record.Service.RepositoryName,
		RepositoryURL:      record.Service.RepositoryURL,
		DefaultBranch:      record.Service.DefaultBranch,
		RootDirectory:      record.Service.RootDirectory,
		DeployStrategy:     record.Service.DeployStrategy,
		DeployTarget:       record.Service.DeployTarget,
		DockerfilePath:     record.Service.DockerfilePath,
		ComposeFile:        record.Service.ComposeFile,
		BuildCommand:       record.Service.BuildCommand,
		DeployCommand:      record.Service.DeployCommand,
		HealthCheckURL:     record.Service.HealthCheckURL,
		Domain:             record.Service.Domain,
		PublishedPort:      record.Service.PublishedPort,
		ContainerPort:      record.Service.ContainerPort,
		AutoDeployEnabled:  record.Service.AutoDeployEnabled,
		WebhookSecret:      record.Service.WebhookSecret,
		LastDeploymentID:   record.Service.LastDeploymentID,
		LastDeployError:    record.Service.LastDeployError,
		CreatedAt:          record.Service.CreatedAt,
		UpdatedAt:          record.Service.UpdatedAt,
	}

	for _, envRow := range record.Environment {
		view.Environment = append(view.Environment, envRow.toView())
	}

	if record.Service.LastDeploymentID != "" {
		deployment, err := s.manager.GetDeployment(record.Service.LastDeploymentID)
		if err == nil {
			snapshot := snapshotFromDeployment(deployment)
			view.LastDeployment = &snapshot
		}
	}

	return view, nil
}

func (s *service) prepareReleaseWorkspace(ctx context.Context, record *serviceRecord, token, branch, commit string) (string, string, error) {
	releaseID := uuid.NewString()
	releaseRoot := filepath.Join(s.workspaceRoot, fmt.Sprintf("service-%d", record.Service.ID), releaseID)
	sourceRoot := filepath.Join(releaseRoot, "repo")
	if err := os.MkdirAll(filepath.Dir(sourceRoot), 0o755); err != nil {
		return "", "", err
	}

	cloneURL, err := injectToken(record.Service.RepositoryURL, token)
	if err != nil {
		return "", "", err
	}

	if err := runSilentCommand(ctx, filepath.Dir(sourceRoot), "git", "clone", "--depth", "1", "--branch", branch, cloneURL, sourceRoot); err != nil {
		return "", "", err
	}

	if strings.TrimSpace(commit) != "" {
		if err := runSilentCommand(ctx, sourceRoot, "git", "fetch", "--depth", "1", "origin", commit); err != nil {
			return "", "", err
		}
		if err := runSilentCommand(ctx, sourceRoot, "git", "checkout", "FETCH_HEAD"); err != nil {
			return "", "", err
		}
	}

	return releaseRoot, sourceRoot, nil
}

func (s *service) writeEnvironmentFile(workingDir string, record *serviceRecord) (string, map[string]string, error) {
	envMap := make(map[string]string, len(record.Environment)+6)
	for _, item := range record.Environment {
		envMap[item.Key] = item.Value
	}
	envMap["HYPERSHIP_PROJECT"] = record.Project.Slug
	envMap["HYPERSHIP_SERVICE"] = record.Service.Slug
	envMap["HYPERSHIP_SERVICE_ID"] = strconv.FormatUint(uint64(record.Service.ID), 10)

	lines := make([]string, 0, len(envMap))
	keys := make([]string, 0, len(envMap))
	for key := range envMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("%s=%s", key, envMap[key]))
	}

	envFile := filepath.Join(workingDir, ".hypership.env")
	if err := os.WriteFile(envFile, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		return "", nil, err
	}

	return envFile, envMap, nil
}

func (s *service) buildPlan(record *serviceRecord, target *deploycontrol.TargetConfig, releaseRoot, workingDir, envFile string, envMap map[string]string, branch, commit, triggeredBy string) (deploycontrol.PlanRequest, error) {
	targetEnv := mergeStringMaps(target.Environment, envMap)
	targetEnv["HYPERSHIP_RELEASE_DIR"] = releaseRoot
	targetEnv["HYPERSHIP_WORKING_DIR"] = workingDir
	targetEnv["HYPERSHIP_ENV_FILE"] = envFile

	plan := deploycontrol.PlanRequest{
		Target:             record.Service.DeployTarget,
		Name:               record.Service.Name,
		TargetName:         record.Service.Name,
		Provider:           firstNonEmpty(target.Provider, "shell"),
		WorkingDirectory:   workingDir,
		HealthCheckURL:     firstNonEmpty(record.Service.HealthCheckURL, inferHealthCheckURL(record.Service)),
		HealthCheckTimeout: firstNonEmpty(target.HealthCheckTimeout, "45s"),
		Environment:        targetEnv,
		Domain:             firstNonEmpty(record.Service.Domain, target.Domain, record.Project.ProductionDomain),
		CertificateMode:    target.CertificateMode,
		TriggeredBy:        firstNonEmpty(triggeredBy, "platform"),
		TriggerMode:        "platform",
		Branch:             branch,
		Commit:             commit,
		Labels: map[string]string{
			"project_id": strconv.FormatUint(uint64(record.Project.ID), 10),
			"service_id": strconv.FormatUint(uint64(record.Service.ID), 10),
			"service":    record.Service.Slug,
			"project":    record.Project.Slug,
			"strategy":   record.Service.DeployStrategy,
		},
	}

	switch record.Service.DeployStrategy {
	case "dockerfile":
		plan.BuildCommand = buildDockerBuildCommand(record.Service)
		plan.DeployCommand = buildDockerRunCommand(record.Service, filepath.Base(envFile))
	case "compose":
		plan.DeployCommand = buildDockerComposeCommand(record.Service, filepath.Base(envFile))
	case "custom":
		plan.BuildCommand = strings.TrimSpace(record.Service.BuildCommand)
		plan.DeployCommand = strings.TrimSpace(record.Service.DeployCommand)
	default:
		return deploycontrol.PlanRequest{}, fmt.Errorf("unsupported strategy %s", record.Service.DeployStrategy)
	}

	if strings.TrimSpace(plan.BuildCommand) == "" && strings.TrimSpace(plan.DeployCommand) == "" {
		return deploycontrol.PlanRequest{}, fmt.Errorf("service %s has no executable deployment commands", record.Service.Name)
	}

	return plan, nil
}

func (s *service) encryptToken(token string) (string, error) {
	return tokenCipher().EncryptString(token)
}

func (s *service) decryptToken(ciphertext string) (string, error) {
	return tokenCipher().DecryptString(ciphertext)
}

func validateServiceConfig(service *ServicePO) error {
	switch normalizeStrategy(service.DeployStrategy) {
	case "dockerfile":
		if service.ContainerPort <= 0 || service.PublishedPort <= 0 {
			return fmt.Errorf("dockerfile strategy requires containerPort and publishedPort")
		}
	case "compose":
		if strings.TrimSpace(firstNonEmpty(service.ComposeFile, "docker-compose.yml")) == "" {
			return fmt.Errorf("compose strategy requires a compose file")
		}
	case "custom":
		if strings.TrimSpace(service.BuildCommand) == "" && strings.TrimSpace(service.DeployCommand) == "" {
			return fmt.Errorf("custom strategy requires buildCommand or deployCommand")
		}
	default:
		return fmt.Errorf("unsupported deploy strategy: %s", service.DeployStrategy)
	}
	return nil
}

func buildEnvRows(serviceID uint, inputs []ServiceEnvironmentVariableInput) []ServiceEnvVarPO {
	rows := make([]ServiceEnvVarPO, 0, len(inputs))
	for _, input := range inputs {
		key := strings.TrimSpace(input.Key)
		if key == "" {
			continue
		}
		rows = append(rows, ServiceEnvVarPO{
			ServiceID: serviceID,
			Key:       key,
			Value:     input.Value,
			IsSecret:  input.IsSecret,
		})
	}
	return rows
}

func buildDockerBuildCommand(service ServicePO) string {
	dockerfilePath := firstNonEmpty(strings.TrimSpace(service.DockerfilePath), "Dockerfile")
	imageName := dockerImageName(service)
	return fmt.Sprintf("docker build -f %s -t %s .", shellQuote(dockerfilePath), shellQuote(imageName))
}

func buildDockerRunCommand(service ServicePO, envFileName string) string {
	containerName := dockerContainerName(service)
	imageName := dockerImageName(service)
	portArg := fmt.Sprintf("-p %d:%d", service.PublishedPort, service.ContainerPort)
	return fmt.Sprintf(
		"docker rm -f %s >/dev/null 2>&1 || true && docker run -d --name %s --restart unless-stopped --env-file %s %s %s",
		shellQuote(containerName),
		shellQuote(containerName),
		shellQuote(envFileName),
		portArg,
		shellQuote(imageName),
	)
}

func buildDockerComposeCommand(service ServicePO, envFileName string) string {
	composeFile := firstNonEmpty(strings.TrimSpace(service.ComposeFile), "docker-compose.yml")
	projectName := "hypership-" + service.Slug
	return fmt.Sprintf(
		"docker compose -f %s --project-name %s --env-file %s up -d --build",
		shellQuote(composeFile),
		shellQuote(projectName),
		shellQuote(envFileName),
	)
}

func dockerImageName(service ServicePO) string {
	return "hypership-" + service.Slug + ":latest"
}

func dockerContainerName(service ServicePO) string {
	return "hypership-" + service.Slug
}

func snapshotFromDeployment(deployment *deploycontrol.Deployment) ServiceDeploymentSnapshot {
	if deployment == nil {
		return ServiceDeploymentSnapshot{}
	}
	return ServiceDeploymentSnapshot{
		ID:         deployment.ID,
		Status:     string(deployment.Status),
		CreatedAt:  deployment.CreatedAt,
		StartedAt:  deployment.StartedAt,
		FinishedAt: deployment.FinishedAt,
		LastLog:    deployment.LastLog,
		Error:      deployment.Error,
	}
}

func injectToken(repositoryURL, token string) (string, error) {
	if strings.TrimSpace(token) == "" {
		return repositoryURL, nil
	}
	parsed, err := filepathToURL(strings.TrimSpace(repositoryURL))
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "file" || parsed.Scheme == "" {
		return repositoryURL, nil
	}
	parsed.User = urlUserPassword("x-access-token", token)
	return parsed.String(), nil
}

func filepathToURL(value string) (*url.URL, error) {
	parsed, err := url.Parse(value)
	if err != nil {
		return nil, err
	}
	return parsed, nil
}

func urlUserPassword(username, password string) *url.Userinfo {
	return url.UserPassword(username, password)
}

func runSilentCommand(ctx context.Context, workdir string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workdir
	output, err := cmd.CombinedOutput()
	if err != nil {
		if len(output) == 0 {
			return err
		}
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func resolveWorkingDirectory(sourceRoot, rootDir string) (string, error) {
	rootDir = strings.TrimSpace(rootDir)
	if rootDir == "" || rootDir == "." {
		return sourceRoot, nil
	}
	resolved := filepath.Clean(filepath.Join(sourceRoot, rootDir))
	sourceRoot = filepath.Clean(sourceRoot)
	if !strings.HasPrefix(resolved, sourceRoot) {
		return "", fmt.Errorf("root directory escapes repository root")
	}
	return resolved, nil
}

func inferHealthCheckURL(service ServicePO) string {
	if service.PublishedPort > 0 {
		return fmt.Sprintf("http://127.0.0.1:%d", service.PublishedPort)
	}
	return ""
}

func mergeStringMaps(base, extra map[string]string) map[string]string {
	if len(base) == 0 && len(extra) == 0 {
		return nil
	}
	result := make(map[string]string, len(base)+len(extra))
	for key, value := range base {
		result[key] = value
	}
	for key, value := range extra {
		result[key] = value
	}
	return result
}

func normalizeStrategy(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "docker", "dockerfile":
		return "dockerfile"
	case "compose", "docker-compose":
		return "compose"
	case "custom", "command":
		return "custom"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = slugPattern.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "app"
	}
	return value
}

func maskToken(token string) string {
	token = strings.TrimSpace(token)
	if len(token) <= 8 {
		return "********"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

func randomSecret() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")
}

func splitCSV(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func currentUTCPtr() *time.Time {
	now := time.Now().UTC()
	return &now
}

func tokenCipher() *cryptocap.AESEncryptor {
	key := firstNonEmpty(env.Get("APP_KEY", ""), env.Get("JWT_SECRET", ""), "hypership-dev-platform-key")
	return cryptocap.NewAESEncryptorFromString(key)
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func verifyGitHubSignature(signature string, payload []byte, secret string) bool {
	signature = strings.TrimSpace(signature)
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}

	expected := hmac.New(sha256.New, []byte(secret))
	expected.Write(payload)
	digest := hex.EncodeToString(expected.Sum(nil))
	return hmac.Equal([]byte("sha256="+digest), []byte(signature))
}
