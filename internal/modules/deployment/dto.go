package deployment

import "github.com/zgiai/zgo/internal/infra/deploycontrol"

type RunDeploymentRequest struct {
	Target      string            `json:"target" binding:"required"`
	Branch      string            `json:"branch,omitempty"`
	Commit      string            `json:"commit,omitempty"`
	TriggeredBy string            `json:"triggeredBy,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
}

type RunDeploymentResponse struct {
	Deployment *deploycontrol.Deployment `json:"deployment"`
}

type GenerateCertificateRequest struct {
	Domain    string `json:"domain" binding:"required"`
	ValidDays int    `json:"validDays,omitempty"`
}

type TriggerWebhookRequest struct {
	Branch      string `json:"branch" binding:"required"`
	Commit      string `json:"commit,omitempty"`
	TriggeredBy string `json:"triggeredBy,omitempty"`
}
