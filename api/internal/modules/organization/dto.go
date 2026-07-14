package organization

import (
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/zgiai/luas/api/internal/domain"
)

var organizationSlugPattern = regexp.MustCompile("^[a-z0-9](?:[a-z0-9-]{1,48}[a-z0-9])$")

// CreateOrganizationRequest creates one organization and owner membership.
type CreateOrganizationRequest struct {
	Name string `json:"name" binding:"required,min=2,max=100"`
	Slug string `json:"slug" binding:"omitempty,min=3,max=50"`
}

// UpdateOrganizationRequest changes mutable organization settings.
type UpdateOrganizationRequest struct {
	Name string `json:"name" binding:"required,min=2,max=100"`
}

// OrganizationResponse is a membership-scoped organization view.
type OrganizationResponse struct {
	ID        uint                    `json:"id"`
	Name      string                  `json:"name"`
	Slug      string                  `json:"slug"`
	Role      domain.OrganizationRole `json:"role"`
	CreatedAt time.Time               `json:"created_at"`
	UpdatedAt time.Time               `json:"updated_at"`
}

func (r *CreateOrganizationRequest) validationErrors() map[string][]string {
	errors := make(map[string][]string)
	if r == nil || !validOrganizationName(r.Name) {
		errors["name"] = []string{"name must contain between 2 and 100 characters"}
	}
	if r != nil && r.Slug != "" && !validOrganizationSlug(r.Slug) {
		errors["slug"] = []string{"slug must contain 3-50 lowercase letters, numbers, or hyphens and cannot end with a hyphen"}
	}
	return errors
}

func (r *UpdateOrganizationRequest) validationErrors() map[string][]string {
	if r != nil && validOrganizationName(r.Name) {
		return nil
	}
	return map[string][]string{
		"name": {"name must contain between 2 and 100 characters"},
	}
}

func validOrganizationName(name string) bool {
	length := utf8.RuneCountInString(strings.TrimSpace(name))
	return length >= 2 && length <= 100
}

func validOrganizationSlug(slug string) bool {
	return organizationSlugPattern.MatchString(slug)
}

func toResponse(membership *domain.OrganizationMembership) *OrganizationResponse {
	if membership == nil || membership.Organization == nil {
		return nil
	}
	organization := membership.Organization
	return &OrganizationResponse{
		ID:        organization.ID,
		Name:      organization.Name,
		Slug:      organization.Slug,
		Role:      membership.Role,
		CreatedAt: organization.CreatedAt,
		UpdatedAt: organization.UpdatedAt,
	}
}
