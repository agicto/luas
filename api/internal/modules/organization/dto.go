package organization

import (
	"net/mail"
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

// CreateOrganizationInvitationRequest offers an organization-scoped role to one email address.
type CreateOrganizationInvitationRequest struct {
	Email string                  `json:"email" binding:"required,max=100"`
	Role  domain.OrganizationRole `json:"role" binding:"required"`
}

// AcceptOrganizationInvitationRequest consumes one opaque invitation token.
type AcceptOrganizationInvitationRequest struct {
	Token string `json:"token" binding:"required,max=256"`
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

// InvitationEmailSendStatus describes only the synchronous provider request.
type InvitationEmailSendStatus string

const (
	InvitationEmailSendStatusAcceptedByProvider InvitationEmailSendStatus = "accepted_by_provider"
	InvitationEmailSendStatusFailed             InvitationEmailSendStatus = "failed"
	InvitationEmailSendStatusNotConfigured      InvitationEmailSendStatus = "not_configured"
)

// OrganizationInvitationResponse is a token-free invitation history view.
type OrganizationInvitationResponse struct {
	ID             uint                                `json:"id"`
	OrganizationID uint                                `json:"organization_id"`
	Email          string                              `json:"email"`
	Role           domain.OrganizationRole             `json:"role"`
	Status         domain.OrganizationInvitationStatus `json:"status"`
	ExpiresAt      time.Time                           `json:"expires_at"`
	CreatedAt      time.Time                           `json:"created_at"`
	UpdatedAt      time.Time                           `json:"updated_at"`
}

// CreateOrganizationInvitationResponse keeps resource state separate from the email attempt.
type CreateOrganizationInvitationResponse struct {
	Invitation      *OrganizationInvitationResponse `json:"invitation"`
	EmailSendStatus InvitationEmailSendStatus       `json:"email_send_status"`
}

// OrganizationInvitationResult is the service result before transport mapping.
type OrganizationInvitationResult struct {
	Invitation      *domain.OrganizationInvitation
	EmailSendStatus InvitationEmailSendStatus
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

func (r *CreateOrganizationInvitationRequest) validationErrors() map[string][]string {
	errors := make(map[string][]string)
	if r == nil || !validInvitationEmailInput(r.Email) {
		errors["email"] = []string{"email must be a valid address with at most 100 characters"}
	}
	if r == nil || !r.Role.CanBeInvited() {
		errors["role"] = []string{"role must be admin or member"}
	}
	return errors
}

func (r *AcceptOrganizationInvitationRequest) validationErrors() map[string][]string {
	if r != nil && strings.TrimSpace(r.Token) != "" {
		return nil
	}
	return map[string][]string{
		"token": {"token is required"},
	}
}

func validOrganizationName(name string) bool {
	length := utf8.RuneCountInString(strings.TrimSpace(name))
	return length >= 2 && length <= 100
}

func validOrganizationSlug(slug string) bool {
	return organizationSlugPattern.MatchString(slug)
}

func normalizeInvitationEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func validInvitationEmailInput(value string) bool {
	normalized := normalizeInvitationEmail(value)
	if normalized == "" || len(normalized) > 100 {
		return false
	}
	address, err := mail.ParseAddress(normalized)
	return err == nil && address.Address == normalized
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

func toInvitationResponse(invitation *domain.OrganizationInvitation, now time.Time) *OrganizationInvitationResponse {
	if invitation == nil {
		return nil
	}
	return &OrganizationInvitationResponse{
		ID:             invitation.ID,
		OrganizationID: invitation.OrganizationID,
		Email:          invitation.Email,
		Role:           invitation.Role,
		Status:         invitation.Status(now),
		ExpiresAt:      invitation.ExpiresAt,
		CreatedAt:      invitation.CreatedAt,
		UpdatedAt:      invitation.UpdatedAt,
	}
}
