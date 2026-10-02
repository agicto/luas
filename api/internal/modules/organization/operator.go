package organization

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/pkg/handler"
	"github.com/zgiai/luas/api/pkg/pagination"
	"github.com/zgiai/luas/api/pkg/response"
)

// OperatorOrganizationResponse is the platform-operator view of one organization.
type OperatorOrganizationResponse struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	CreatedBy   uint      `json:"created_by"`
	MemberCount int64     `json:"member_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// OperatorMemberResponse is the platform-operator view of one membership. Unlike the tenant member
// view it includes the email, which operators already see in the operator user list.
type OperatorMemberResponse struct {
	ID       uint                    `json:"id"`
	UserID   uint                    `json:"user_id"`
	Username string                  `json:"username"`
	Nickname string                  `json:"nickname"`
	Email    string                  `json:"email"`
	Role     domain.OrganizationRole `json:"role"`
	JoinedAt time.Time               `json:"joined_at"`
}

type operatorStore interface {
	operatorList(ctx context.Context, query string, page, pageSize int) ([]OperatorOrganizationResponse, int64, error)
	operatorFind(ctx context.Context, organizationID uint) (*OperatorOrganizationResponse, error)
	operatorMembers(ctx context.Context, organizationID uint, page, pageSize int) ([]OperatorMemberResponse, int64, error)
}

// OperatorHandler serves platform-wide organization reads to the starter that owns operator
// authorization. It applies no membership check and must only be mounted behind that authorization.
type OperatorHandler struct {
	store operatorStore
}

// NewOperatorHandler creates the operator-facing organization read handler.
func NewOperatorHandler(repo *repository) *OperatorHandler {
	return &OperatorHandler{store: repo}
}

type operatorListQuery struct {
	Query string `form:"q" binding:"max=100"`
}

// List returns every organization, newest first, optionally filtered by name or slug.
func (h *OperatorHandler) List(c *gin.Context) {
	var query operatorListQuery
	if !handler.BindQuery(c, &query) {
		return
	}
	page := pagination.FromContext(c)
	items, total, err := h.store.operatorList(
		c.Request.Context(),
		strings.TrimSpace(query.Query),
		page.GetPage(),
		page.GetPerPage(),
	)
	if err != nil {
		response.HandleError(c, "Failed to list organizations", err)
		return
	}
	paginator := pagination.NewPaginator(items, total, page.GetPage(), page.GetPerPage())
	paginator.SetPath(c.Request.URL.Path).WithQuery(c.Request.URL.Query())
	response.Success(c, paginator)
}

// Require aborts with ORGANIZATION.NOT_FOUND unless the path organization exists. Starters whose
// operator routes are nested under an organization mount behind it.
func (h *OperatorHandler) Require(c *gin.Context) {
	organizationID, ok := handler.ParseID(c, "id")
	if !ok {
		c.Abort()
		return
	}
	if _, err := h.store.operatorFind(c.Request.Context(), organizationID); err != nil {
		response.HandleError(c, "Failed to load organization", err)
		c.Abort()
		return
	}
	c.Next()
}

// Get returns one organization by ID.
func (h *OperatorHandler) Get(c *gin.Context) {
	organizationID, ok := handler.ParseID(c, "id")
	if !ok {
		return
	}
	organization, err := h.store.operatorFind(c.Request.Context(), organizationID)
	if err != nil {
		response.HandleError(c, "Failed to load organization", err)
		return
	}
	response.Success(c, organization)
}

// ListMembers returns the members of one organization ordered by user ID.
func (h *OperatorHandler) ListMembers(c *gin.Context) {
	organizationID, ok := handler.ParseID(c, "id")
	if !ok {
		return
	}
	page := pagination.FromContext(c)
	items, total, err := h.store.operatorMembers(c.Request.Context(), organizationID, page.GetPage(), page.GetPerPage())
	if err != nil {
		response.HandleError(c, "Failed to list organization members", err)
		return
	}
	paginator := pagination.NewPaginator(items, total, page.GetPage(), page.GetPerPage())
	paginator.SetPath(c.Request.URL.Path)
	response.Success(c, paginator)
}

func (r *repository) operatorList(
	ctx context.Context,
	query string,
	page, pageSize int,
) ([]OperatorOrganizationResponse, int64, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, 0, err
	}
	if page < 1 || pageSize < 1 || pageSize > 100 {
		return nil, 0, domain.ErrInvalidInput
	}
	scope := db.Model(&OrganizationPO{})
	if query != "" {
		pattern := "%" + escapeOperatorLike(strings.ToLower(query)) + "%"
		scope = scope.Where(`(LOWER(name) LIKE ? ESCAPE '\' OR LOWER(slug) LIKE ? ESCAPE '\')`, pattern, pattern)
	}
	var total int64
	if countErr := scope.Count(&total).Error; countErr != nil {
		return nil, 0, countErr
	}
	var rows []OrganizationPO
	if findErr := scope.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; findErr != nil {
		return nil, 0, findErr
	}
	ids := make([]uint, len(rows))
	for index := range rows {
		ids[index] = rows[index].ID
	}
	counts, err := operatorMemberCounts(db, ids)
	if err != nil {
		return nil, 0, err
	}
	items := make([]OperatorOrganizationResponse, len(rows))
	for index := range rows {
		items[index] = newOperatorOrganizationResponse(&rows[index], counts[rows[index].ID])
	}
	return items, total, nil
}

func (r *repository) operatorFind(ctx context.Context, organizationID uint) (*OperatorOrganizationResponse, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}
	if organizationID == 0 {
		return nil, domain.ErrInvalidInput
	}
	var row OrganizationPO
	if findErr := db.First(&row, organizationID).Error; findErr != nil {
		if errors.Is(findErr, gorm.ErrRecordNotFound) {
			return nil, domain.ErrOrganizationNotFound
		}
		return nil, findErr
	}
	counts, err := operatorMemberCounts(db, []uint{row.ID})
	if err != nil {
		return nil, err
	}
	result := newOperatorOrganizationResponse(&row, counts[row.ID])
	return &result, nil
}

func (r *repository) operatorMembers(
	ctx context.Context,
	organizationID uint,
	page, pageSize int,
) ([]OperatorMemberResponse, int64, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, 0, err
	}
	if organizationID == 0 || page < 1 || pageSize < 1 || pageSize > 100 {
		return nil, 0, domain.ErrInvalidInput
	}
	scope := operatorMemberScope(db).Where("organization_memberships.organization_id = ?", organizationID)
	var total int64
	if err := scope.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []OrganizationMembershipPO
	if err := scope.
		Preload("User").
		Order("organization_memberships.user_id ASC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	items := make([]OperatorMemberResponse, len(rows))
	for index := range rows {
		row := &rows[index]
		items[index] = OperatorMemberResponse{
			ID:       row.ID,
			UserID:   row.UserID,
			Username: row.User.Username,
			Nickname: row.User.Nickname,
			Email:    row.User.Email,
			Role:     domain.OrganizationRole(row.Role),
			JoinedAt: row.CreatedAt,
		}
	}
	return items, total, nil
}

// operatorMemberScope selects memberships whose account still exists, as the tenant member list does.
func operatorMemberScope(db *gorm.DB) *gorm.DB {
	return db.Model(&OrganizationMembershipPO{}).
		Joins("JOIN users AS member_users ON member_users.id = organization_memberships.user_id AND member_users.deleted_at IS NULL")
}

func operatorMemberCounts(db *gorm.DB, organizationIDs []uint) (map[uint]int64, error) {
	counts := make(map[uint]int64, len(organizationIDs))
	if len(organizationIDs) == 0 {
		return counts, nil
	}
	var rows []struct {
		OrganizationID uint
		Members        int64
	}
	if err := operatorMemberScope(db).
		Select("organization_memberships.organization_id AS organization_id, COUNT(*) AS members").
		Where("organization_memberships.organization_id IN ?", organizationIDs).
		Group("organization_memberships.organization_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		counts[row.OrganizationID] = row.Members
	}
	return counts, nil
}

func newOperatorOrganizationResponse(row *OrganizationPO, members int64) OperatorOrganizationResponse {
	return OperatorOrganizationResponse{
		ID:          row.ID,
		Name:        row.Name,
		Slug:        row.Slug,
		CreatedBy:   row.CreatedBy,
		MemberCount: members,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func escapeOperatorLike(value string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(value)
}
