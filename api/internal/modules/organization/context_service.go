package organization

import (
	"context"
	"fmt"

	"github.com/zgiai/luas/api/internal/domain"
)

func (s *service) ResolveContext(
	ctx context.Context,
	userID, organizationID uint,
) (*domain.OrganizationContext, error) {
	if userID == 0 || organizationID == 0 {
		return nil, domain.ErrInvalidInput
	}

	resolved, err := s.repo.ResolveContext(ctx, organizationID, userID)
	if err != nil {
		return nil, fmt.Errorf("resolve organization context: %w", err)
	}
	if resolved == nil || !resolved.IsValid() || resolved.UserID != userID || resolved.OrganizationID != organizationID {
		return nil, domain.ErrServiceUnavailable
	}
	return resolved, nil
}
