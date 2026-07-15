package setting

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/modules/user"
	"github.com/zgiai/luas/api/internal/starter/assembly"
	httphandler "github.com/zgiai/luas/api/pkg/handler"
	"github.com/zgiai/luas/api/pkg/response"
)

const maxSettingMutationBodyBytes = maxSettingValueBytes + 256

var settingVersionETagPattern = regexp.MustCompile(`^"setting-v(0|[1-9][0-9]{0,19})"$`)

// Handler owns public, user, and active-organization setting HTTP behavior.
type Handler struct {
	service        Service
	cleaner        user.AccountDeletionCleaner
	deletionPolicy *user.AccountDeletionPolicy
}

var (
	_ assembly.Module           = (*Handler)(nil)
	_ assembly.RouteModule      = (*Handler)(nil)
	_ assembly.ActivationModule = (*Handler)(nil)
)

// NewHandler creates the optional setting HTTP boundary.
func NewHandler(service *service, deletionPolicy *user.AccountDeletionPolicy) *Handler {
	return &Handler{
		service:        service,
		cleaner:        service,
		deletionPolicy: deletionPolicy,
	}
}

func (h *Handler) Name() string { return "setting" }

// Activate installs user-setting cleanup only when the starter is selected.
func (h *Handler) Activate() error {
	return h.deletionPolicy.RegisterCleaner(h.cleaner)
}

// PublicApp lists cacheable public app settings and honors aggregate revalidation.
func (h *Handler) PublicApp(c *gin.Context) {
	values, err := h.service.ListPublicAppSettings(c.Request.Context())
	if err != nil {
		setSettingNoStore(c)
		response.HandleError(c, "Failed to load public settings", err)
		return
	}
	payload := toSettingResponses(values)
	etag, err := aggregateSettingETag(payload)
	if err != nil {
		setSettingNoStore(c)
		response.HandleError(c, "Failed to encode public settings", domain.ErrServiceUnavailable)
		return
	}
	c.Header("Cache-Control", "public, max-age=60, stale-while-revalidate=300")
	c.Header("ETag", etag)
	if ifNoneMatch(c.GetHeader("If-None-Match"), etag) {
		c.Status(http.StatusNotModified)
		return
	}
	response.Success(c, payload)
}

// UserList returns the current user's private effective settings.
func (h *Handler) UserList(c *gin.Context) {
	setPrivateSettingResponse(c, false)
	userID, ok := httphandler.GetUserID(c)
	if !ok {
		return
	}
	h.list(c, domain.SettingTarget{Scope: domain.SettingScopeUser, SubjectID: userID})
}

// UserSet creates or replaces one current-user override.
func (h *Handler) UserSet(c *gin.Context) {
	setPrivateSettingResponse(c, false)
	userID, ok := httphandler.GetUserID(c)
	if !ok {
		return
	}
	h.set(c, domain.SettingTarget{Scope: domain.SettingScopeUser, SubjectID: userID})
}

// UserReset resets one current-user setting to its code default.
func (h *Handler) UserReset(c *gin.Context) {
	setPrivateSettingResponse(c, false)
	userID, ok := httphandler.GetUserID(c)
	if !ok {
		return
	}
	h.reset(c, domain.SettingTarget{Scope: domain.SettingScopeUser, SubjectID: userID})
}

// OrganizationList returns settings for the verified active organization.
func (h *Handler) OrganizationList(c *gin.Context) {
	setPrivateSettingResponse(c, true)
	organization, ok := settingOrganizationContext(c)
	if !ok {
		return
	}
	h.list(c, domain.SettingTarget{
		Scope:     domain.SettingScopeOrganization,
		SubjectID: organization.OrganizationID,
	})
}

// OrganizationSet replaces an organization override for an owner/admin.
func (h *Handler) OrganizationSet(c *gin.Context) {
	setPrivateSettingResponse(c, true)
	organization, ok := settingOrganizationContext(c)
	if !ok {
		return
	}
	if !organization.Role.CanManageOrganization() {
		response.HandleError(c, "Organization setting update forbidden", domain.ErrPermissionDenied)
		return
	}
	h.set(c, domain.SettingTarget{
		Scope:     domain.SettingScopeOrganization,
		SubjectID: organization.OrganizationID,
	})
}

// OrganizationReset resets an organization override for an owner/admin.
func (h *Handler) OrganizationReset(c *gin.Context) {
	setPrivateSettingResponse(c, true)
	organization, ok := settingOrganizationContext(c)
	if !ok {
		return
	}
	if !organization.Role.CanManageOrganization() {
		response.HandleError(c, "Organization setting reset forbidden", domain.ErrPermissionDenied)
		return
	}
	h.reset(c, domain.SettingTarget{
		Scope:     domain.SettingScopeOrganization,
		SubjectID: organization.OrganizationID,
	})
}

// luas:bounded-list max=64 reason=finite-code-owned-catalog
func (h *Handler) list(c *gin.Context, target domain.SettingTarget) {
	values, err := h.service.ListSettings(c.Request.Context(), target)
	if err != nil {
		response.HandleError(c, "Failed to load settings", err)
		return
	}
	response.Success(c, toSettingResponses(values))
}

func (h *Handler) set(c *gin.Context, target domain.SettingTarget) {
	expectedVersion, ok := expectedSettingVersion(c)
	if !ok {
		return
	}
	request, ok := bindSettingMutation(c)
	if !ok {
		return
	}
	value, err := h.service.SetSetting(
		c.Request.Context(),
		target,
		c.Param("key"),
		request.Value,
		expectedVersion,
	)
	if err != nil {
		response.HandleError(c, "Failed to update setting", err)
		return
	}
	c.Header("ETag", settingVersionETag(value.Version))
	response.Success(c, toSettingResponse(value))
}

func (h *Handler) reset(c *gin.Context, target domain.SettingTarget) {
	expectedVersion, ok := expectedSettingVersion(c)
	if !ok {
		return
	}
	version, err := h.service.ResetSetting(
		c.Request.Context(),
		target,
		c.Param("key"),
		expectedVersion,
	)
	if err != nil {
		response.HandleError(c, "Failed to reset setting", err)
		return
	}
	c.Header("ETag", settingVersionETag(version))
	response.NoContent(c)
}

func expectedSettingVersion(c *gin.Context) (uint64, bool) {
	values := c.Request.Header.Values("If-Match")
	if len(values) == 0 {
		response.HandleError(c, "Setting version precondition required", domain.ErrSettingPreconditionRequired)
		return 0, false
	}
	if len(values) != 1 || strings.Contains(values[0], ",") {
		response.ErrorWithCode(c, http.StatusBadRequest, domain.CodeInvalidInput, "Invalid If-Match header")
		return 0, false
	}
	match := settingVersionETagPattern.FindStringSubmatch(values[0])
	if len(match) != 2 {
		response.ErrorWithCode(c, http.StatusBadRequest, domain.CodeInvalidInput, "Invalid If-Match header")
		return 0, false
	}
	version, err := strconv.ParseUint(match[1], 10, 64)
	if err != nil {
		response.ErrorWithCode(c, http.StatusBadRequest, domain.CodeInvalidInput, "Invalid If-Match header")
		return 0, false
	}
	return version, true
}

func bindSettingMutation(c *gin.Context) (*setSettingRequest, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSettingMutationBodyBytes)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	var request setSettingRequest
	if err := decoder.Decode(&request); err != nil {
		if strings.HasPrefix(err.Error(), "json: unknown field ") {
			response.HandleError(c, "Invalid setting value", domain.ErrSettingInvalidValue)
			return nil, false
		}
		response.BadRequest(c, "Invalid setting request", err)
		return nil, false
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		response.BadRequest(c, "Invalid setting request", fmt.Errorf("request must contain exactly one JSON object"))
		return nil, false
	}
	if request.Value == nil {
		response.HandleError(c, "Invalid setting value", domain.ErrSettingInvalidValue)
		return nil, false
	}
	return &request, true
}

func settingOrganizationContext(c *gin.Context) (domain.OrganizationContext, bool) {
	value, ok := domain.OrganizationContextFromContext(c.Request.Context())
	if !ok {
		response.HandleError(c, "Organization context required", domain.ErrOrganizationContextRequired)
		return domain.OrganizationContext{}, false
	}
	return value, true
}

func aggregateSettingETag(values []*SettingResponse) (string, error) {
	encoded, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return `"settings-` + hex.EncodeToString(digest[:]) + `"`, nil
}

func settingVersionETag(version uint64) string {
	return `"setting-v` + strconv.FormatUint(version, 10) + `"`
}

func ifNoneMatch(header string, etag string) bool {
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || candidate == etag || strings.TrimPrefix(candidate, "W/") == etag {
			return true
		}
	}
	return false
}

func setPrivateSettingResponse(c *gin.Context, organization bool) {
	c.Header("Cache-Control", "private, no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Vary", "Authorization")
	if organization {
		c.Header("Vary", "Authorization, Organization-Id")
	}
}

func setSettingNoStore(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
}
