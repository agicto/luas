package platform

import "github.com/zgiai/zgo/internal/infra/deploycontrol"

type ConnectGitHubRequest struct {
	Name  string `json:"name,omitempty"`
	Token string `json:"token" binding:"required"`
}

type CreateProjectRequest struct {
	Name             string `json:"name" binding:"required"`
	Description      string `json:"description,omitempty"`
	ProductionDomain string `json:"productionDomain,omitempty"`
}

type ServiceEnvironmentVariableInput struct {
	Key      string `json:"key" binding:"required"`
	Value    string `json:"value"`
	IsSecret bool   `json:"isSecret"`
}

type ImportServiceRequest struct {
	ProjectID               uint                              `json:"projectId,omitempty"`
	ProjectName             string                            `json:"projectName,omitempty"`
	ProjectDescription      string                            `json:"projectDescription,omitempty"`
	ProjectProductionDomain string                            `json:"projectProductionDomain,omitempty"`
	GitHubConnectionID      uint                              `json:"githubConnectionId" binding:"required"`
	Name                    string                            `json:"name,omitempty"`
	RepositoryOwner         string                            `json:"repositoryOwner" binding:"required"`
	RepositoryName          string                            `json:"repositoryName" binding:"required"`
	RepositoryURL           string                            `json:"repositoryUrl,omitempty"`
	Branch                  string                            `json:"branch" binding:"required"`
	RootDirectory           string                            `json:"rootDirectory,omitempty"`
	DeployStrategy          string                            `json:"deployStrategy" binding:"required"`
	DeployTarget            string                            `json:"deployTarget" binding:"required"`
	DockerfilePath          string                            `json:"dockerfilePath,omitempty"`
	ComposeFile             string                            `json:"composeFile,omitempty"`
	BuildCommand            string                            `json:"buildCommand,omitempty"`
	DeployCommand           string                            `json:"deployCommand,omitempty"`
	HealthCheckURL          string                            `json:"healthCheckUrl,omitempty"`
	Domain                  string                            `json:"domain,omitempty"`
	PublishedPort           int                               `json:"publishedPort,omitempty"`
	ContainerPort           int                               `json:"containerPort,omitempty"`
	AutoDeployEnabled       bool                              `json:"autoDeployEnabled,omitempty"`
	Environment             []ServiceEnvironmentVariableInput `json:"environment,omitempty"`
}

type UpdateServiceRequest struct {
	Name              string `json:"name,omitempty"`
	Branch            string `json:"branch,omitempty"`
	RootDirectory     string `json:"rootDirectory,omitempty"`
	DeployStrategy    string `json:"deployStrategy,omitempty"`
	DeployTarget      string `json:"deployTarget,omitempty"`
	DockerfilePath    string `json:"dockerfilePath,omitempty"`
	ComposeFile       string `json:"composeFile,omitempty"`
	BuildCommand      string `json:"buildCommand,omitempty"`
	DeployCommand     string `json:"deployCommand,omitempty"`
	HealthCheckURL    string `json:"healthCheckUrl,omitempty"`
	Domain            string `json:"domain,omitempty"`
	PublishedPort     int    `json:"publishedPort,omitempty"`
	ContainerPort     int    `json:"containerPort,omitempty"`
	AutoDeployEnabled *bool  `json:"autoDeployEnabled,omitempty"`
}

type ReplaceEnvironmentRequest struct {
	Variables []ServiceEnvironmentVariableInput `json:"variables"`
}

type TriggerServiceDeploymentRequest struct {
	Branch      string `json:"branch,omitempty"`
	Commit      string `json:"commit,omitempty"`
	TriggeredBy string `json:"triggeredBy,omitempty"`
}

type ImportServiceResponse struct {
	Service *Service `json:"service"`
}

type TriggerServiceDeploymentResponse struct {
	Service    *Service                  `json:"service"`
	Deployment *deploycontrol.Deployment `json:"deployment"`
}
