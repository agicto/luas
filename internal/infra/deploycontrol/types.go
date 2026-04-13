package deploycontrol

import "time"

type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
)

type CertificateMode string

const (
	CertificateModeDisabled      CertificateMode = "disabled"
	CertificateModeSelfSigned    CertificateMode = "self-signed"
	CertificateModeRenderManaged CertificateMode = "render-managed"
)

type TargetConfig struct {
	Name               string            `json:"name"`
	DisplayName        string            `json:"displayName"`
	Provider           string            `json:"provider"`
	WorkingDirectory   string            `json:"workingDirectory"`
	BuildCommand       string            `json:"buildCommand,omitempty"`
	DeployCommand      string            `json:"deployCommand"`
	HealthCheckURL     string            `json:"healthCheckUrl,omitempty"`
	HealthCheckTimeout string            `json:"healthCheckTimeout,omitempty"`
	Environment        map[string]string `json:"environment,omitempty"`
	Domain             string            `json:"domain,omitempty"`
	CertificateMode    CertificateMode   `json:"certificateMode,omitempty"`
	AutoDeployBranches []string          `json:"autoDeployBranches,omitempty"`
	WebhookSecret      string            `json:"webhookSecret,omitempty"`
}

type TargetList struct {
	Targets []TargetConfig `json:"targets"`
}

type RunRequest struct {
	Target      string            `json:"target"`
	Branch      string            `json:"branch,omitempty"`
	Commit      string            `json:"commit,omitempty"`
	TriggeredBy string            `json:"triggeredBy,omitempty"`
	TriggerMode string            `json:"triggerMode,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
}

type PlanRequest struct {
	Target             string            `json:"target,omitempty"`
	Name               string            `json:"name"`
	TargetName         string            `json:"targetName,omitempty"`
	Provider           string            `json:"provider,omitempty"`
	WorkingDirectory   string            `json:"workingDirectory"`
	BuildCommand       string            `json:"buildCommand,omitempty"`
	DeployCommand      string            `json:"deployCommand,omitempty"`
	HealthCheckURL     string            `json:"healthCheckUrl,omitempty"`
	HealthCheckTimeout string            `json:"healthCheckTimeout,omitempty"`
	Environment        map[string]string `json:"environment,omitempty"`
	Domain             string            `json:"domain,omitempty"`
	CertificateMode    CertificateMode   `json:"certificateMode,omitempty"`
	TriggeredBy        string            `json:"triggeredBy,omitempty"`
	TriggerMode        string            `json:"triggerMode,omitempty"`
	Branch             string            `json:"branch,omitempty"`
	Commit             string            `json:"commit,omitempty"`
	Labels             map[string]string `json:"labels,omitempty"`
}

type Deployment struct {
	ID                 string            `json:"id"`
	Target             string            `json:"target"`
	TargetName         string            `json:"targetName"`
	Provider           string            `json:"provider"`
	Status             Status            `json:"status"`
	TriggeredBy        string            `json:"triggeredBy"`
	TriggerMode        string            `json:"triggerMode"`
	Branch             string            `json:"branch,omitempty"`
	Commit             string            `json:"commit,omitempty"`
	WorkingDirectory   string            `json:"workingDirectory"`
	Command            string            `json:"command"`
	Environment        map[string]string `json:"environment,omitempty"`
	Labels             map[string]string `json:"labels,omitempty"`
	Domain             string            `json:"domain,omitempty"`
	HealthCheckURL     string            `json:"healthCheckUrl,omitempty"`
	HealthCheckTimeout string            `json:"healthCheckTimeout,omitempty"`
	CertificateMode    CertificateMode   `json:"certificateMode"`
	Certificate        *CertificateInfo  `json:"certificate,omitempty"`
	LastLog            string            `json:"lastLog,omitempty"`
	Error              string            `json:"error,omitempty"`
	CreatedAt          time.Time         `json:"createdAt"`
	StartedAt          *time.Time        `json:"startedAt,omitempty"`
	FinishedAt         *time.Time        `json:"finishedAt,omitempty"`
}

type LogEntry struct {
	Sequence     int64     `json:"sequence"`
	DeploymentID string    `json:"deploymentId"`
	Timestamp    time.Time `json:"timestamp"`
	Stream       string    `json:"stream"`
	Message      string    `json:"message"`
}

type CertificateRequest struct {
	Domain    string `json:"domain"`
	ValidDays int    `json:"validDays,omitempty"`
}

type CertificateInfo struct {
	Domain      string    `json:"domain"`
	CertPath    string    `json:"certPath"`
	KeyPath     string    `json:"keyPath"`
	GeneratedAt time.Time `json:"generatedAt"`
	ExpiresAt   time.Time `json:"expiresAt"`
	Mode        string    `json:"mode"`
}

type WatchEvent struct {
	Deployment *Deployment `json:"deployment,omitempty"`
	Log        *LogEntry   `json:"log,omitempty"`
	Done       bool        `json:"done"`
}
