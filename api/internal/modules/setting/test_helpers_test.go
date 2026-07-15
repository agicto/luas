package setting

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/infra/database"
	"github.com/zgiai/luas/api/internal/modules/organization"
	"github.com/zgiai/luas/api/internal/modules/user"
)

type settingTestFixture struct {
	db           *gorm.DB
	service      *service
	repository   *repository
	user         *user.UserPO
	organization *organization.OrganizationPO
}

func newSettingTestFixture(t *testing.T) settingTestFixture {
	t.Helper()
	db, err := database.NewTestDB()
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&user.UserPO{},
		&organization.OrganizationPO{},
		&SettingPO{},
	))
	owner := &user.UserPO{
		Username: "setting-owner",
		Email:    "setting-owner@example.test",
		Password: "hashed-password",
		Status:   1,
	}
	require.NoError(t, db.Create(owner).Error)
	organizationPO := &organization.OrganizationPO{
		Name:      "Setting Organization",
		Slug:      "setting-organization",
		CreatedBy: owner.ID,
	}
	require.NoError(t, db.Create(organizationPO).Error)
	catalog, err := NewDefaultCatalog()
	require.NoError(t, err)
	repository := NewRepository(db)
	service := NewService(
		catalog,
		repository,
		&config.Config{Starters: config.StarterConfig{Optional: []string{"organization", "setting"}}},
	)
	return settingTestFixture{
		db:           db,
		service:      service,
		repository:   repository,
		user:         owner,
		organization: organizationPO,
	}
}
