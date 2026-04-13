package platform

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

var errRecordNotFound = errors.New("platform: record not found")

type serviceRecord struct {
	Service     ServicePO
	Project     ProjectPO
	Connection  GitHubConnectionPO
	Environment []ServiceEnvVarPO
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *repository {
	return &repository{db: db}
}

func (r *repository) UpsertGitHubConnection(ctx context.Context, connection *GitHubConnectionPO) error {
	var existing GitHubConnectionPO
	err := r.db.WithContext(ctx).Where("login = ?", connection.Login).First(&existing).Error
	if err == nil {
		existing.Name = connection.Name
		existing.Provider = connection.Provider
		existing.DisplayName = connection.DisplayName
		existing.AvatarURL = connection.AvatarURL
		existing.TokenEncrypted = connection.TokenEncrypted
		existing.TokenMasked = connection.TokenMasked
		existing.Scopes = connection.Scopes
		existing.LastSyncedAt = connection.LastSyncedAt
		if saveErr := r.db.WithContext(ctx).Save(&existing).Error; saveErr != nil {
			return saveErr
		}
		connection.ID = existing.ID
		connection.CreatedAt = existing.CreatedAt
		connection.UpdatedAt = existing.UpdatedAt
		return nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return r.db.WithContext(ctx).Create(connection).Error
}

func (r *repository) ListGitHubConnections(ctx context.Context) ([]GitHubConnectionPO, error) {
	var rows []GitHubConnectionPO
	if err := r.db.WithContext(ctx).Order("updated_at desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *repository) GetGitHubConnection(ctx context.Context, id uint) (*GitHubConnectionPO, error) {
	var row GitHubConnectionPO
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errRecordNotFound
		}
		return nil, err
	}
	return &row, nil
}

func (r *repository) CreateProject(ctx context.Context, project *ProjectPO) error {
	return r.db.WithContext(ctx).Create(project).Error
}

func (r *repository) UpdateProject(ctx context.Context, project *ProjectPO) error {
	return r.db.WithContext(ctx).Save(project).Error
}

func (r *repository) GetProject(ctx context.Context, id uint) (*ProjectPO, error) {
	var row ProjectPO
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errRecordNotFound
		}
		return nil, err
	}
	return &row, nil
}

func (r *repository) ListProjects(ctx context.Context) ([]ProjectPO, map[uint]int, error) {
	var projects []ProjectPO
	if err := r.db.WithContext(ctx).Order("updated_at desc").Find(&projects).Error; err != nil {
		return nil, nil, err
	}

	type countRow struct {
		ProjectID uint
		Count     int
	}

	var counts []countRow
	if err := r.db.WithContext(ctx).
		Model(&ServicePO{}).
		Select("project_id, count(*) as count").
		Group("project_id").
		Scan(&counts).Error; err != nil {
		return nil, nil, err
	}

	serviceCount := make(map[uint]int, len(counts))
	for _, row := range counts {
		serviceCount[row.ProjectID] = row.Count
	}

	return projects, serviceCount, nil
}

func (r *repository) CountProjects(ctx context.Context) (int64, error) {
	return r.countByModel(ctx, &ProjectPO{})
}

func (r *repository) CountServices(ctx context.Context) (int64, error) {
	return r.countByModel(ctx, &ServicePO{})
}

func (r *repository) CountGitHubConnections(ctx context.Context) (int64, error) {
	return r.countByModel(ctx, &GitHubConnectionPO{})
}

func (r *repository) CreateService(ctx context.Context, service *ServicePO) error {
	return r.db.WithContext(ctx).Create(service).Error
}

func (r *repository) UpdateService(ctx context.Context, service *ServicePO) error {
	return r.db.WithContext(ctx).Save(service).Error
}

func (r *repository) ListServices(ctx context.Context) ([]serviceRecord, error) {
	var rows []ServicePO
	if err := r.db.WithContext(ctx).Order("updated_at desc").Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]serviceRecord, 0, len(rows))
	for _, row := range rows {
		record, err := r.GetServiceRecord(ctx, row.ID, true)
		if err != nil {
			return nil, err
		}
		result = append(result, *record)
	}

	return result, nil
}

func (r *repository) GetServiceRecord(ctx context.Context, id uint, withEnv bool) (*serviceRecord, error) {
	var service ServicePO
	if err := r.db.WithContext(ctx).First(&service, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errRecordNotFound
		}
		return nil, err
	}

	project, err := r.GetProject(ctx, service.ProjectID)
	if err != nil {
		return nil, err
	}

	connection, err := r.GetGitHubConnection(ctx, service.GitHubConnectionID)
	if err != nil {
		return nil, err
	}

	record := &serviceRecord{
		Service:    service,
		Project:    *project,
		Connection: *connection,
	}

	if withEnv {
		envRows, envErr := r.ListServiceEnvVars(ctx, service.ID)
		if envErr != nil {
			return nil, envErr
		}
		record.Environment = envRows
	}

	return record, nil
}

func (r *repository) ListServiceEnvVars(ctx context.Context, serviceID uint) ([]ServiceEnvVarPO, error) {
	var rows []ServiceEnvVarPO
	if err := r.db.WithContext(ctx).
		Where("service_id = ?", serviceID).
		Order("key asc").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *repository) ReplaceServiceEnvVars(ctx context.Context, serviceID uint, values []ServiceEnvVarPO) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("service_id = ?", serviceID).Delete(&ServiceEnvVarPO{}).Error; err != nil {
			return err
		}
		if len(values) == 0 {
			return nil
		}
		return tx.Create(&values).Error
	})
}

func (r *repository) ProjectSlugExists(ctx context.Context, slug string) (bool, error) {
	return r.slugExists(ctx, &ProjectPO{}, slug)
}

func (r *repository) ServiceSlugExists(ctx context.Context, slug string) (bool, error) {
	return r.slugExists(ctx, &ServicePO{}, slug)
}

func (r *repository) countByModel(ctx context.Context, model any) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(model).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *repository) slugExists(ctx context.Context, model any, slug string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(model).Where("slug = ?", strings.TrimSpace(slug)).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *repository) MustGetServiceRecord(ctx context.Context, id uint) (*serviceRecord, error) {
	record, err := r.GetServiceRecord(ctx, id, true)
	if err != nil {
		return nil, fmt.Errorf("failed to load service %d: %w", id, err)
	}
	return record, nil
}
