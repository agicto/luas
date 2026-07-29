package usage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/config"
	testplatform "github.com/zgiai/luas/api/internal/infra/testing"
	"github.com/zgiai/luas/api/internal/modules/organization"
	"github.com/zgiai/luas/api/internal/modules/user"
)

type usageTestFixture struct {
	db           *gorm.DB
	service      *service
	repository   *repository
	user         *user.UserPO
	organization *organization.OrganizationPO
	now          time.Time
}

func newUsageTestFixture(t *testing.T) usageTestFixture {
	t.Helper()
	db := testplatform.OpenPostgres(t, nil,
		&user.UserPO{},
		&user.AuthenticationSessionPO{},
		&organization.OrganizationPO{},
		&UsageCounterPO{},
		&UsageQuotaPO{},
		&UsageEventPO{},
	)
	owner := &user.UserPO{
		Username: "usage-owner",
		Email:    "usage-owner@example.test",
		Password: "hashed-password",
		Status:   1,
	}
	require.NoError(t, db.Create(owner).Error)
	organizationPO := &organization.OrganizationPO{
		Name:      "Usage Organization",
		Slug:      "usage-organization",
		CreatedBy: owner.ID,
	}
	require.NoError(t, db.Create(organizationPO).Error)
	catalog, err := NewDefaultCatalog()
	require.NoError(t, err)
	repository := NewRepository(db)
	service := NewService(
		catalog,
		repository,
		&config.Config{Starters: config.StarterConfig{Optional: []string{"organization", "usage"}}},
	)
	now := time.Date(2026, time.July, 15, 12, 30, 0, 0, time.UTC)
	repository.now = func() time.Time { return now }
	service.now = func() time.Time { return now }
	return usageTestFixture{
		db:           db,
		service:      service,
		repository:   repository,
		user:         owner,
		organization: organizationPO,
		now:          now,
	}
}
