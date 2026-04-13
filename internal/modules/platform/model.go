package platform

import (
	"time"

	"gorm.io/gorm"
)

type GitHubConnectionPO struct {
	ID             uint `gorm:"primaryKey"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
	Name           string         `gorm:"size:120;not null"`
	Provider       string         `gorm:"size:40;not null;default:github"`
	Login          string         `gorm:"size:120;not null;uniqueIndex"`
	DisplayName    string         `gorm:"size:160"`
	AvatarURL      string         `gorm:"size:512"`
	TokenEncrypted string         `gorm:"type:text;not null"`
	TokenMasked    string         `gorm:"size:24;not null"`
	Scopes         string         `gorm:"size:255"`
	LastSyncedAt   *time.Time
}

func (GitHubConnectionPO) TableName() string {
	return "platform_github_connections"
}

type ProjectPO struct {
	ID               uint `gorm:"primaryKey"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt `gorm:"index"`
	Name             string         `gorm:"size:120;not null"`
	Slug             string         `gorm:"size:120;not null;uniqueIndex"`
	Description      string         `gorm:"size:500"`
	ProductionDomain string         `gorm:"size:255"`
}

func (ProjectPO) TableName() string {
	return "platform_projects"
}

type ServicePO struct {
	ID                 uint `gorm:"primaryKey"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeletedAt          gorm.DeletedAt `gorm:"index"`
	ProjectID          uint           `gorm:"index;not null"`
	GitHubConnectionID uint           `gorm:"index;not null"`
	Name               string         `gorm:"size:120;not null"`
	Slug               string         `gorm:"size:120;not null;uniqueIndex"`
	RepositoryOwner    string         `gorm:"size:120;not null"`
	RepositoryName     string         `gorm:"size:120;not null"`
	RepositoryURL      string         `gorm:"size:255;not null"`
	DefaultBranch      string         `gorm:"size:120;not null"`
	RootDirectory      string         `gorm:"size:255"`
	DeployStrategy     string         `gorm:"size:40;not null"`
	DeployTarget       string         `gorm:"size:120;not null"`
	DockerfilePath     string         `gorm:"size:255"`
	ComposeFile        string         `gorm:"size:255"`
	BuildCommand       string         `gorm:"type:text"`
	DeployCommand      string         `gorm:"type:text"`
	HealthCheckURL     string         `gorm:"size:500"`
	Domain             string         `gorm:"size:255"`
	PublishedPort      int
	ContainerPort      int
	AutoDeployEnabled  bool   `gorm:"not null;default:false"`
	WebhookSecret      string `gorm:"size:120"`
	LastDeploymentID   string `gorm:"size:64"`
	LastDeployError    string `gorm:"type:text"`
}

func (ServicePO) TableName() string {
	return "platform_services"
}

type ServiceEnvVarPO struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	ServiceID uint           `gorm:"index;not null"`
	Key       string         `gorm:"size:160;not null"`
	Value     string         `gorm:"type:text;not null"`
	IsSecret  bool           `gorm:"not null;default:true"`
}

func (ServiceEnvVarPO) TableName() string {
	return "platform_service_env_vars"
}

type GitHubConnection struct {
	ID           uint       `json:"id"`
	Name         string     `json:"name"`
	Provider     string     `json:"provider"`
	Login        string     `json:"login"`
	DisplayName  string     `json:"displayName"`
	AvatarURL    string     `json:"avatarUrl"`
	TokenMasked  string     `json:"tokenMasked"`
	Scopes       []string   `json:"scopes,omitempty"`
	LastSyncedAt *time.Time `json:"lastSyncedAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

type GitHubViewer struct {
	Login       string
	DisplayName string
	AvatarURL   string
}

type GitHubRepository struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	FullName      string    `json:"fullName"`
	Owner         string    `json:"owner"`
	DefaultBranch string    `json:"defaultBranch"`
	CloneURL      string    `json:"cloneUrl"`
	HTMLURL       string    `json:"htmlUrl"`
	Private       bool      `json:"private"`
	Description   string    `json:"description,omitempty"`
	Language      string    `json:"language,omitempty"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type Project struct {
	ID               uint      `json:"id"`
	Name             string    `json:"name"`
	Slug             string    `json:"slug"`
	Description      string    `json:"description,omitempty"`
	ProductionDomain string    `json:"productionDomain,omitempty"`
	ServiceCount     int       `json:"serviceCount"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type ServiceEnvironmentVariable struct {
	ID       uint   `json:"id"`
	Key      string `json:"key"`
	Value    string `json:"value"`
	IsSecret bool   `json:"isSecret"`
}

type Service struct {
	ID                 uint                         `json:"id"`
	ProjectID          uint                         `json:"projectId"`
	ProjectName        string                       `json:"projectName"`
	GitHubConnectionID uint                         `json:"githubConnectionId"`
	GitHubLogin        string                       `json:"githubLogin"`
	Name               string                       `json:"name"`
	Slug               string                       `json:"slug"`
	RepositoryOwner    string                       `json:"repositoryOwner"`
	RepositoryName     string                       `json:"repositoryName"`
	RepositoryURL      string                       `json:"repositoryUrl"`
	DefaultBranch      string                       `json:"defaultBranch"`
	RootDirectory      string                       `json:"rootDirectory,omitempty"`
	DeployStrategy     string                       `json:"deployStrategy"`
	DeployTarget       string                       `json:"deployTarget"`
	DockerfilePath     string                       `json:"dockerfilePath,omitempty"`
	ComposeFile        string                       `json:"composeFile,omitempty"`
	BuildCommand       string                       `json:"buildCommand,omitempty"`
	DeployCommand      string                       `json:"deployCommand,omitempty"`
	HealthCheckURL     string                       `json:"healthCheckUrl,omitempty"`
	Domain             string                       `json:"domain,omitempty"`
	PublishedPort      int                          `json:"publishedPort,omitempty"`
	ContainerPort      int                          `json:"containerPort,omitempty"`
	AutoDeployEnabled  bool                         `json:"autoDeployEnabled"`
	WebhookSecret      string                       `json:"webhookSecret,omitempty"`
	LastDeploymentID   string                       `json:"lastDeploymentId,omitempty"`
	LastDeployment     *ServiceDeploymentSnapshot   `json:"lastDeployment,omitempty"`
	LastDeployError    string                       `json:"lastDeployError,omitempty"`
	Environment        []ServiceEnvironmentVariable `json:"environment,omitempty"`
	CreatedAt          time.Time                    `json:"createdAt"`
	UpdatedAt          time.Time                    `json:"updatedAt"`
}

type ServiceDeploymentSnapshot struct {
	ID         string     `json:"id"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"createdAt"`
	StartedAt  *time.Time `json:"startedAt,omitempty"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
	LastLog    string     `json:"lastLog,omitempty"`
	Error      string     `json:"error,omitempty"`
}

type PlatformOverview struct {
	Projects          int                         `json:"projects"`
	Services          int                         `json:"services"`
	GitHubConnections int                         `json:"githubConnections"`
	RecentDeployments int                         `json:"recentDeployments"`
	ProjectsList      []Project                   `json:"projectsList"`
	ServicesList      []Service                   `json:"servicesList"`
	Deployments       []ServiceDeploymentSnapshot `json:"deployments"`
}

func (po *GitHubConnectionPO) toView() GitHubConnection {
	return GitHubConnection{
		ID:           po.ID,
		Name:         po.Name,
		Provider:     po.Provider,
		Login:        po.Login,
		DisplayName:  po.DisplayName,
		AvatarURL:    po.AvatarURL,
		TokenMasked:  po.TokenMasked,
		Scopes:       splitCSV(po.Scopes),
		LastSyncedAt: po.LastSyncedAt,
		CreatedAt:    po.CreatedAt,
		UpdatedAt:    po.UpdatedAt,
	}
}

func (po *ProjectPO) toView(serviceCount int) Project {
	return Project{
		ID:               po.ID,
		Name:             po.Name,
		Slug:             po.Slug,
		Description:      po.Description,
		ProductionDomain: po.ProductionDomain,
		ServiceCount:     serviceCount,
		CreatedAt:        po.CreatedAt,
		UpdatedAt:        po.UpdatedAt,
	}
}

func (po *ServiceEnvVarPO) toView() ServiceEnvironmentVariable {
	return ServiceEnvironmentVariable{
		ID:       po.ID,
		Key:      po.Key,
		Value:    po.Value,
		IsSecret: po.IsSecret,
	}
}
