package deployment

import (
	"context"
	"crypto/subtle"
	"fmt"
	"slices"
	"strings"

	"github.com/zgiai/zgo/internal/infra/deploycontrol"
)

type Service interface {
	ListTargets() ([]deploycontrol.TargetConfig, error)
	ListDeployments(limit int) ([]*deploycontrol.Deployment, error)
	GetDeployment(id string) (*deploycontrol.Deployment, error)
	ListLogs(id string, tail int) ([]deploycontrol.LogEntry, error)
	StartDeployment(ctx context.Context, req deploycontrol.RunRequest) (*deploycontrol.Deployment, error)
	Watch(id string) (<-chan deploycontrol.WatchEvent, func(), error)
	GenerateCertificate(req deploycontrol.CertificateRequest) (*deploycontrol.CertificateInfo, error)
	HandleWebhook(ctx context.Context, targetName, secret string, req TriggerWebhookRequest) (*deploycontrol.Deployment, error)
}

type service struct {
	manager *deploycontrol.Manager
}

func NewService(manager *deploycontrol.Manager) Service {
	return &service{manager: manager}
}

func (s *service) ListTargets() ([]deploycontrol.TargetConfig, error) {
	return s.manager.ListTargets()
}

func (s *service) ListDeployments(limit int) ([]*deploycontrol.Deployment, error) {
	return s.manager.ListDeployments(limit)
}

func (s *service) GetDeployment(id string) (*deploycontrol.Deployment, error) {
	return s.manager.GetDeployment(id)
}

func (s *service) ListLogs(id string, tail int) ([]deploycontrol.LogEntry, error) {
	return s.manager.ListLogs(id, tail)
}

func (s *service) StartDeployment(ctx context.Context, req deploycontrol.RunRequest) (*deploycontrol.Deployment, error) {
	_ = ctx
	return s.manager.StartDeployment(req)
}

func (s *service) Watch(id string) (<-chan deploycontrol.WatchEvent, func(), error) {
	return s.manager.Watch(id)
}

func (s *service) GenerateCertificate(req deploycontrol.CertificateRequest) (*deploycontrol.CertificateInfo, error) {
	return s.manager.GenerateCertificate(req)
}

func (s *service) HandleWebhook(ctx context.Context, targetName, secret string, req TriggerWebhookRequest) (*deploycontrol.Deployment, error) {
	target, err := s.manager.GetTarget(targetName)
	if err != nil {
		return nil, err
	}

	if target.WebhookSecret != "" && subtle.ConstantTimeCompare([]byte(target.WebhookSecret), []byte(secret)) != 1 {
		return nil, fmt.Errorf("invalid webhook secret")
	}

	if len(target.AutoDeployBranches) > 0 && !slices.Contains(target.AutoDeployBranches, req.Branch) {
		return nil, fmt.Errorf("branch %s is not enabled for auto deploy", req.Branch)
	}

	return s.StartDeployment(ctx, deploycontrol.RunRequest{
		Target:      targetName,
		Branch:      strings.TrimSpace(req.Branch),
		Commit:      strings.TrimSpace(req.Commit),
		TriggeredBy: defaultValue(strings.TrimSpace(req.TriggeredBy), "webhook"),
		TriggerMode: "webhook",
	})
}

func defaultValue(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
