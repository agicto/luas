package organization

import (
	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/pkg/handler"
	"github.com/zgiai/luas/api/pkg/pagination"
	"github.com/zgiai/luas/api/pkg/response"
)

// ListMembers returns the PII-minimized directory visible to organization members.
func (h *Handler) ListMembers(c *gin.Context) {
	userID, ok := handler.GetUserID(c)
	if !ok {
		return
	}
	organizationID, ok := handler.ParseID(c, "id")
	if !ok {
		return
	}

	page := pagination.FromContext(c)
	memberships, total, err := h.service.ListMembers(
		c.Request.Context(),
		userID,
		organizationID,
		page.GetPage(),
		page.GetPerPage(),
	)
	if err != nil {
		response.HandleError(c, "Failed to list organization members", err)
		return
	}

	items := make([]*OrganizationMemberResponse, len(memberships))
	for index, membership := range memberships {
		items[index] = toMemberResponse(membership)
	}
	paginator := pagination.NewPaginator(items, total, page.GetPage(), page.GetPerPage())
	paginator.SetPath(c.Request.URL.Path)
	response.Success(c, paginator)
}

// ChangeMemberRole grants or revokes admin without changing ownership.
func (h *Handler) ChangeMemberRole(c *gin.Context) {
	userID, ok := handler.GetUserID(c)
	if !ok {
		return
	}
	organizationID, ok := handler.ParseID(c, "id")
	if !ok {
		return
	}
	memberID, ok := handler.ParseID(c, "member_id")
	if !ok {
		return
	}

	var req UpdateOrganizationMemberRequest
	if !handler.BindJSON(c, &req) {
		return
	}
	if errors := req.validationErrors(); len(errors) > 0 {
		response.ValidationFailed(c, errors)
		return
	}

	membership, err := h.service.ChangeMemberRole(c.Request.Context(), userID, organizationID, memberID, &req)
	if err != nil {
		response.HandleError(c, "Failed to change organization member role", err)
		return
	}
	response.Success(c, toMemberResponse(membership))
}

// RemoveMember removes another member or lets a non-owner leave.
func (h *Handler) RemoveMember(c *gin.Context) {
	userID, ok := handler.GetUserID(c)
	if !ok {
		return
	}
	organizationID, ok := handler.ParseID(c, "id")
	if !ok {
		return
	}
	memberID, ok := handler.ParseID(c, "member_id")
	if !ok {
		return
	}

	if err := h.service.RemoveMember(c.Request.Context(), userID, organizationID, memberID); err != nil {
		response.HandleError(c, "Failed to remove organization member", err)
		return
	}
	response.NoContent(c)
}

// TransferOwnership atomically promotes one existing member and demotes the caller.
func (h *Handler) TransferOwnership(c *gin.Context) {
	userID, ok := handler.GetUserID(c)
	if !ok {
		return
	}
	organizationID, ok := handler.ParseID(c, "id")
	if !ok {
		return
	}

	var req TransferOrganizationOwnershipRequest
	if !handler.BindJSON(c, &req) {
		return
	}
	if errors := req.validationErrors(); len(errors) > 0 {
		response.ValidationFailed(c, errors)
		return
	}

	transfer, err := h.service.TransferOwnership(c.Request.Context(), userID, organizationID, &req)
	if err != nil {
		response.HandleError(c, "Failed to transfer organization ownership", err)
		return
	}
	response.Success(c, &OrganizationOwnershipTransferResponse{
		PreviousOwner: toMemberResponse(transfer.PreviousOwner),
		NewOwner:      toMemberResponse(transfer.NewOwner),
	})
}
