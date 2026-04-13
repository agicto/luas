package deploycontrol

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/zgiai/zgo/pkg/env"
)

var (
	ErrTargetNotFound     = errors.New("deployment target not found")
	ErrDeploymentNotFound = errors.New("deployment not found")
)

type Manager struct {
	configPath  string
	configDir   string
	storageRoot string

	mu               sync.RWMutex
	subscriptions    map[string]map[int]chan WatchEvent
	nextSubscription int
}

func NewManager() *Manager {
	configPath := env.Get("DEPLOY_TARGETS_PATH", "deploy.targets.json")
	storageRoot := env.Get("DEPLOY_STORAGE_ROOT", filepath.Join("storage", "deployments"))

	manager := &Manager{
		configPath:       configPath,
		configDir:        filepath.Dir(configPath),
		storageRoot:      storageRoot,
		subscriptions:    make(map[string]map[int]chan WatchEvent),
		nextSubscription: 1,
	}

	_ = os.MkdirAll(manager.runsDir(), 0o755)
	_ = os.MkdirAll(manager.logsDir(), 0o755)
	_ = os.MkdirAll(manager.certsDir(), 0o755)

	return manager
}

func (m *Manager) ListTargets() ([]TargetConfig, error) {
	cfg, err := m.loadConfig()
	if err != nil {
		return nil, err
	}

	targets := append([]TargetConfig(nil), cfg.Targets...)
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].DisplayName < targets[j].DisplayName
	})

	return targets, nil
}

func (m *Manager) GetTarget(name string) (*TargetConfig, error) {
	cfg, err := m.loadConfig()
	if err != nil {
		return nil, err
	}

	for _, target := range cfg.Targets {
		if target.Name == name {
			resolved := target
			resolved.WorkingDirectory = m.resolveWorkingDirectory(target.WorkingDirectory)
			if resolved.CertificateMode == "" {
				resolved.CertificateMode = m.defaultCertificateMode(resolved)
			}
			if resolved.Provider == "" {
				resolved.Provider = "shell"
			}
			return &resolved, nil
		}
	}

	return nil, ErrTargetNotFound
}

func (m *Manager) StartDeployment(req RunRequest) (*Deployment, error) {
	target, err := m.GetTarget(req.Target)
	if err != nil {
		return nil, err
	}

	deployment, err := m.prepareDeployment(specFromTargetRequest(req, *target))
	if err != nil {
		return nil, err
	}

	go func() {
		_, _ = m.runDeployment(context.Background(), deployment, target.BuildCommand, target.DeployCommand, nil)
	}()

	return deployment, nil
}

func (m *Manager) RunDeployment(ctx context.Context, req RunRequest, watcher func(WatchEvent)) (*Deployment, error) {
	target, err := m.GetTarget(req.Target)
	if err != nil {
		return nil, err
	}

	deployment, err := m.prepareDeployment(specFromTargetRequest(req, *target))
	if err != nil {
		return nil, err
	}

	return m.runDeployment(ctx, deployment, target.BuildCommand, target.DeployCommand, watcher)
}

func (m *Manager) StartPlan(req PlanRequest) (*Deployment, error) {
	deployment, err := m.prepareDeployment(specFromPlanRequest(req))
	if err != nil {
		return nil, err
	}

	buildCommand := strings.TrimSpace(req.BuildCommand)
	deployCommand := strings.TrimSpace(req.DeployCommand)

	go func() {
		_, _ = m.runDeployment(context.Background(), deployment, buildCommand, deployCommand, nil)
	}()

	return deployment, nil
}

func (m *Manager) RunPlan(ctx context.Context, req PlanRequest, watcher func(WatchEvent)) (*Deployment, error) {
	deployment, err := m.prepareDeployment(specFromPlanRequest(req))
	if err != nil {
		return nil, err
	}

	return m.runDeployment(ctx, deployment, strings.TrimSpace(req.BuildCommand), strings.TrimSpace(req.DeployCommand), watcher)
}

func (m *Manager) ListDeployments(limit int) ([]*Deployment, error) {
	if limit <= 0 {
		limit = 20
	}

	entries, err := os.ReadDir(m.runsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return []*Deployment{}, nil
		}
		return nil, err
	}

	deployments := make([]*Deployment, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		deployment, err := m.loadDeployment(strings.TrimSuffix(entry.Name(), ".json"))
		if err != nil {
			continue
		}
		deployments = append(deployments, deployment)
	}

	sort.Slice(deployments, func(i, j int) bool {
		return deployments[i].CreatedAt.After(deployments[j].CreatedAt)
	})

	if len(deployments) > limit {
		deployments = deployments[:limit]
	}

	return deployments, nil
}

func (m *Manager) GetDeployment(id string) (*Deployment, error) {
	return m.loadDeployment(id)
}

func (m *Manager) ListDeploymentsByLabel(key, value string, limit int) ([]*Deployment, error) {
	deployments, err := m.ListDeployments(0)
	if err != nil {
		return nil, err
	}

	filtered := make([]*Deployment, 0, len(deployments))
	for _, deployment := range deployments {
		if deployment == nil || deployment.Labels == nil {
			continue
		}
		if deployment.Labels[key] != value {
			continue
		}
		filtered = append(filtered, deployment)
	}

	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}

	return filtered, nil
}

func (m *Manager) ListLogs(id string, tail int) ([]LogEntry, error) {
	path := m.logFile(id)
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []LogEntry{}, nil
		}
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []LogEntry{}, nil
	}

	if tail > 0 && len(lines) > tail {
		lines = lines[len(lines)-tail:]
	}

	entries := make([]LogEntry, 0, len(lines))
	for _, line := range lines {
		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err == nil {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

func (m *Manager) Watch(id string) (<-chan WatchEvent, func(), error) {
	deployment, err := m.loadDeployment(id)
	if err != nil {
		return nil, nil, err
	}

	events := make(chan WatchEvent, 64)
	var cancelOnce sync.Once

	logs, err := m.ListLogs(id, 200)
	if err == nil {
		for i := range logs {
			entry := logs[i]
			events <- WatchEvent{Log: &entry}
		}
	}

	events <- WatchEvent{Deployment: deployment, Done: deployment.FinishedAt != nil}

	m.mu.Lock()
	subscriptionID := m.nextSubscription
	m.nextSubscription++
	if m.subscriptions[id] == nil {
		m.subscriptions[id] = make(map[int]chan WatchEvent)
	}
	m.subscriptions[id][subscriptionID] = events
	m.mu.Unlock()

	cancel := func() {
		cancelOnce.Do(func() {
			m.mu.Lock()
			defer m.mu.Unlock()

			if subs := m.subscriptions[id]; subs != nil {
				delete(subs, subscriptionID)
				if len(subs) == 0 {
					delete(m.subscriptions, id)
				}
			}
			close(events)
		})
	}

	if deployment.FinishedAt != nil {
		go cancel()
	}

	return events, cancel, nil
}

func (m *Manager) GenerateCertificate(req CertificateRequest) (*CertificateInfo, error) {
	domain := strings.TrimSpace(req.Domain)
	if domain == "" {
		return nil, fmt.Errorf("domain is required")
	}

	validDays := req.ValidDays
	if validDays <= 0 {
		validDays = 90
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	serialNumber, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	expiresAt := now.Add(time.Duration(validDays) * 24 * time.Hour)
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   domain,
			Organization: []string{"Hypership Deploy Control"},
		},
		NotBefore:             now.Add(-5 * time.Minute),
		NotAfter:              expiresAt,
		DNSNames:              []string{domain},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, err
	}

	safeName := sanitizeName(domain)
	certPath := filepath.Join(m.certsDir(), safeName+".crt")
	keyPath := filepath.Join(m.certsDir(), safeName+".key")

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	if err := os.WriteFile(certPath, certPEM, 0o644); err != nil {
		return nil, err
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return nil, err
	}

	return &CertificateInfo{
		Domain:      domain,
		CertPath:    certPath,
		KeyPath:     keyPath,
		GeneratedAt: now,
		ExpiresAt:   expiresAt,
		Mode:        string(CertificateModeSelfSigned),
	}, nil
}

func (m *Manager) prepareDeployment(spec deploymentSpec) (*Deployment, error) {
	if strings.TrimSpace(spec.BuildCommand) == "" && strings.TrimSpace(spec.DeployCommand) == "" {
		return nil, fmt.Errorf("deployment %s has no build or deploy command", spec.TargetName)
	}

	triggerMode := strings.TrimSpace(spec.TriggerMode)
	if triggerMode == "" {
		triggerMode = "manual"
	}

	triggeredBy := strings.TrimSpace(spec.TriggeredBy)
	if triggeredBy == "" {
		triggeredBy = "system"
	}

	deployment := &Deployment{
		ID:                 uuid.NewString(),
		Target:             spec.Target,
		TargetName:         firstNonEmpty(spec.TargetName, spec.Target, "dynamic"),
		Provider:           firstNonEmpty(spec.Provider, "shell"),
		Status:             StatusPending,
		TriggeredBy:        triggeredBy,
		TriggerMode:        triggerMode,
		Branch:             strings.TrimSpace(spec.Branch),
		Commit:             strings.TrimSpace(spec.Commit),
		WorkingDirectory:   spec.WorkingDirectory,
		Command:            strings.TrimSpace(strings.Join([]string{spec.BuildCommand, spec.DeployCommand}, " && ")),
		Environment:        cloneMap(spec.Environment),
		Labels:             cloneMap(spec.Labels),
		Domain:             strings.TrimSpace(spec.Domain),
		HealthCheckURL:     strings.TrimSpace(spec.HealthCheckURL),
		HealthCheckTimeout: strings.TrimSpace(spec.HealthCheckTimeout),
		CertificateMode:    spec.CertificateMode,
		CreatedAt:          time.Now().UTC(),
	}

	if err := m.saveDeployment(deployment); err != nil {
		return nil, err
	}

	return deployment, nil
}

func (m *Manager) runDeployment(ctx context.Context, deployment *Deployment, buildCommand, deployCommand string, watcher func(WatchEvent)) (*Deployment, error) {
	now := time.Now().UTC()
	deployment.Status = StatusRunning
	deployment.StartedAt = &now
	if err := m.saveDeployment(deployment); err != nil {
		return nil, err
	}
	m.broadcast(deployment.ID, WatchEvent{Deployment: cloneDeployment(deployment)})
	m.forward(watcher, WatchEvent{Deployment: cloneDeployment(deployment)})

	m.emitLog(deployment.ID, "system", fmt.Sprintf("Starting deployment for target %s", deployment.TargetName), watcher)
	m.emitLog(deployment.ID, "system", fmt.Sprintf("Working directory: %s", deployment.WorkingDirectory), watcher)

	if err := m.prepareCertificate(deployment, watcher); err != nil {
		return m.failDeployment(deployment, err, watcher)
	}

	steps := []struct {
		label   string
		command string
	}{
		{label: "build", command: strings.TrimSpace(buildCommand)},
		{label: "deploy", command: strings.TrimSpace(deployCommand)},
	}

	for _, step := range steps {
		if step.command == "" {
			continue
		}

		m.emitLog(deployment.ID, "system", fmt.Sprintf("[%s] %s", step.label, step.command), watcher)
		if err := m.executeCommand(ctx, deployment, step.command, watcher); err != nil {
			return m.failDeployment(deployment, err, watcher)
		}
	}

	if err := m.runHealthCheck(deployment, watcher); err != nil {
		return m.failDeployment(deployment, err, watcher)
	}

	finishedAt := time.Now().UTC()
	deployment.Status = StatusSucceeded
	deployment.FinishedAt = &finishedAt
	deployment.Error = ""
	if err := m.saveDeployment(deployment); err != nil {
		return nil, err
	}

	m.emitLog(deployment.ID, "system", "Deployment completed successfully", watcher)
	event := WatchEvent{Deployment: cloneDeployment(deployment), Done: true}
	m.broadcast(deployment.ID, event)
	m.forward(watcher, event)

	return deployment, nil
}

func (m *Manager) failDeployment(deployment *Deployment, err error, watcher func(WatchEvent)) (*Deployment, error) {
	finishedAt := time.Now().UTC()
	deployment.Status = StatusFailed
	deployment.FinishedAt = &finishedAt
	deployment.Error = err.Error()
	if saveErr := m.saveDeployment(deployment); saveErr != nil {
		return nil, saveErr
	}

	m.emitLog(deployment.ID, "stderr", err.Error(), watcher)
	event := WatchEvent{Deployment: cloneDeployment(deployment), Done: true}
	m.broadcast(deployment.ID, event)
	m.forward(watcher, event)

	return deployment, err
}

func (m *Manager) prepareCertificate(deployment *Deployment, watcher func(WatchEvent)) error {
	switch deployment.CertificateMode {
	case CertificateModeDisabled:
		return nil
	case CertificateModeRenderManaged:
		if deployment.Domain != "" {
			m.emitLog(deployment.ID, "system", fmt.Sprintf("TLS will be managed by Render for %s", deployment.Domain), watcher)
		}
		return nil
	case CertificateModeSelfSigned:
		if deployment.Domain == "" {
			return fmt.Errorf("target %s is configured for self-signed certificates but has no domain", deployment.Target)
		}
		cert, err := m.GenerateCertificate(CertificateRequest{Domain: deployment.Domain})
		if err != nil {
			return err
		}
		deployment.Certificate = cert
		if err := m.saveDeployment(deployment); err != nil {
			return err
		}
		m.emitLog(deployment.ID, "system", fmt.Sprintf("Generated self-signed certificate: %s", cert.CertPath), watcher)
		return nil
	default:
		return nil
	}
}

func (m *Manager) executeCommand(ctx context.Context, deployment *Deployment, command string, watcher func(WatchEvent)) error {
	cmd := exec.CommandContext(ctx, "/bin/sh", "-lc", command)
	cmd.Dir = deployment.WorkingDirectory
	cmd.Env = append(os.Environ(), mapToEnv(deployment.Environment)...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go m.streamPipe(&wg, deployment.ID, "stdout", stdout, watcher)
	go m.streamPipe(&wg, deployment.ID, "stderr", stderr, watcher)
	wg.Wait()

	return cmd.Wait()
}

func (m *Manager) streamPipe(wg *sync.WaitGroup, deploymentID, stream string, reader io.Reader, watcher func(WatchEvent)) {
	defer wg.Done()

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		m.emitLog(deploymentID, stream, scanner.Text(), watcher)
	}
}

func (m *Manager) runHealthCheck(deployment *Deployment, watcher func(WatchEvent)) error {
	if deployment.HealthCheckURL == "" {
		return nil
	}

	timeout := 30 * time.Second
	if strings.TrimSpace(deployment.HealthCheckTimeout) != "" {
		if parsed, parseErr := time.ParseDuration(deployment.HealthCheckTimeout); parseErr == nil {
			timeout = parsed
		}
	}

	m.emitLog(deployment.ID, "system", fmt.Sprintf("Running health check: %s", deployment.HealthCheckURL), watcher)

	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, deployment.HealthCheckURL, nil)
		resp, err := http.DefaultClient.Do(req)
		if err == nil && resp != nil && resp.StatusCode >= 200 && resp.StatusCode < 400 {
			_ = resp.Body.Close()
			m.emitLog(deployment.ID, "system", fmt.Sprintf("Health check passed with status %d", resp.StatusCode), watcher)
			return nil
		}
		if resp != nil {
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("health check returned status %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(2 * time.Second)
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("health check timed out")
	}
	return lastErr
}

func (m *Manager) emitLog(deploymentID, stream, message string, watcher func(WatchEvent)) {
	entry := LogEntry{
		Sequence:     time.Now().UnixNano(),
		DeploymentID: deploymentID,
		Timestamp:    time.Now().UTC(),
		Stream:       stream,
		Message:      strings.TrimRight(message, "\n"),
	}

	_ = m.appendLog(entry)

	if deployment, err := m.loadDeployment(deploymentID); err == nil {
		deployment.LastLog = entry.Message
		_ = m.saveDeployment(deployment)
	}

	event := WatchEvent{Log: &entry}
	m.broadcast(deploymentID, event)
	m.forward(watcher, event)
}

func (m *Manager) appendLog(entry LogEntry) error {
	encoded, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	path := m.logFile(entry.DeploymentID)
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(append(encoded, '\n'))
	return err
}

func (m *Manager) saveDeployment(deployment *Deployment) error {
	path := m.runFile(deployment.ID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(deployment, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

func (m *Manager) loadDeployment(id string) (*Deployment, error) {
	path := m.runFile(id)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrDeploymentNotFound
		}
		return nil, err
	}

	var deployment Deployment
	if err := json.Unmarshal(data, &deployment); err != nil {
		return nil, err
	}
	return &deployment, nil
}

func (m *Manager) loadConfig() (*TargetList, error) {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &TargetList{Targets: []TargetConfig{}}, nil
		}
		return nil, err
	}

	var cfg TargetList
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (m *Manager) resolveWorkingDirectory(path string) string {
	if strings.TrimSpace(path) == "" {
		return "."
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Clean(filepath.Join(m.configDir, path))
}

func (m *Manager) defaultCertificateMode(target TargetConfig) CertificateMode {
	if strings.EqualFold(target.Provider, "render") && strings.TrimSpace(target.Domain) != "" {
		return CertificateModeRenderManaged
	}
	return CertificateModeDisabled
}

func (m *Manager) runsDir() string  { return filepath.Join(m.storageRoot, "runs") }
func (m *Manager) logsDir() string  { return filepath.Join(m.storageRoot, "logs") }
func (m *Manager) certsDir() string { return filepath.Join(m.storageRoot, "certs") }

func (m *Manager) runFile(id string) string { return filepath.Join(m.runsDir(), id+".json") }
func (m *Manager) logFile(id string) string { return filepath.Join(m.logsDir(), id+".ndjson") }

func (m *Manager) broadcast(deploymentID string, event WatchEvent) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, ch := range m.subscriptions[deploymentID] {
		select {
		case ch <- event:
		default:
		}
	}
}

func (m *Manager) forward(watcher func(WatchEvent), event WatchEvent) {
	if watcher != nil {
		watcher(event)
	}
}

func mergeEnv(base, extra map[string]string) map[string]string {
	if len(base) == 0 && len(extra) == 0 {
		return nil
	}

	merged := make(map[string]string, len(base)+len(extra))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range extra {
		merged[key] = value
	}
	return merged
}

func mapToEnv(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}

	envList := make([]string, 0, len(values))
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		envList = append(envList, fmt.Sprintf("%s=%s", key, values[key]))
	}
	return envList
}

func cloneDeployment(deployment *Deployment) *Deployment {
	if deployment == nil {
		return nil
	}

	clone := *deployment
	clone.Environment = mergeEnv(deployment.Environment, nil)
	clone.Labels = mergeEnv(deployment.Labels, nil)
	if deployment.Certificate != nil {
		certClone := *deployment.Certificate
		clone.Certificate = &certClone
	}
	return &clone
}

type deploymentSpec struct {
	Target             string
	TargetName         string
	Provider           string
	WorkingDirectory   string
	BuildCommand       string
	DeployCommand      string
	HealthCheckURL     string
	HealthCheckTimeout string
	Environment        map[string]string
	Labels             map[string]string
	Domain             string
	CertificateMode    CertificateMode
	TriggeredBy        string
	TriggerMode        string
	Branch             string
	Commit             string
}

func specFromTargetRequest(req RunRequest, target TargetConfig) deploymentSpec {
	return deploymentSpec{
		Target:             target.Name,
		TargetName:         firstNonEmpty(target.DisplayName, target.Name),
		Provider:           firstNonEmpty(target.Provider, "shell"),
		WorkingDirectory:   target.WorkingDirectory,
		BuildCommand:       strings.TrimSpace(target.BuildCommand),
		DeployCommand:      strings.TrimSpace(target.DeployCommand),
		HealthCheckURL:     strings.TrimSpace(target.HealthCheckURL),
		HealthCheckTimeout: strings.TrimSpace(target.HealthCheckTimeout),
		Environment:        mergeEnv(target.Environment, req.Environment),
		Domain:             strings.TrimSpace(target.Domain),
		CertificateMode:    target.CertificateMode,
		TriggeredBy:        req.TriggeredBy,
		TriggerMode:        req.TriggerMode,
		Branch:             req.Branch,
		Commit:             req.Commit,
	}
}

func specFromPlanRequest(req PlanRequest) deploymentSpec {
	return deploymentSpec{
		Target:             firstNonEmpty(req.Target, req.Name, "dynamic"),
		TargetName:         firstNonEmpty(req.TargetName, req.Name, "Dynamic Deployment"),
		Provider:           firstNonEmpty(req.Provider, "shell"),
		WorkingDirectory:   req.WorkingDirectory,
		BuildCommand:       strings.TrimSpace(req.BuildCommand),
		DeployCommand:      strings.TrimSpace(req.DeployCommand),
		HealthCheckURL:     strings.TrimSpace(req.HealthCheckURL),
		HealthCheckTimeout: strings.TrimSpace(req.HealthCheckTimeout),
		Environment:        cloneMap(req.Environment),
		Labels:             cloneMap(req.Labels),
		Domain:             strings.TrimSpace(req.Domain),
		CertificateMode:    req.CertificateMode,
		TriggeredBy:        req.TriggeredBy,
		TriggerMode:        req.TriggerMode,
		Branch:             req.Branch,
		Commit:             req.Commit,
	}
}

func cloneMap(values map[string]string) map[string]string {
	return mergeEnv(values, nil)
}

func sanitizeName(value string) string {
	replacer := strings.NewReplacer(".", "-", "*", "wildcard", "/", "-", "\\", "-", ":", "-")
	return replacer.Replace(strings.TrimSpace(value))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
