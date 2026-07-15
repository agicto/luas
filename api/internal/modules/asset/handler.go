package asset

import (
	"mime"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/modules/user"
	"github.com/zgiai/luas/api/internal/starter/assembly"
	httphandler "github.com/zgiai/luas/api/pkg/handler"
	"github.com/zgiai/luas/api/pkg/pagination"
	"github.com/zgiai/luas/api/pkg/response"
)

// Handler owns authenticated asset management and token-authenticated local transfers.
type Handler struct {
	service        Service
	deletionGuard  user.AccountDeletionGuard
	deletionPolicy *user.AccountDeletionPolicy
}

var (
	_ assembly.Module           = (*Handler)(nil)
	_ assembly.RouteModule      = (*Handler)(nil)
	_ assembly.ActivationModule = (*Handler)(nil)
)

// NewHandler creates the optional asset HTTP boundary.
func NewHandler(service *service, deletionPolicy *user.AccountDeletionPolicy) *Handler {
	return &Handler{
		service:        service,
		deletionGuard:  service,
		deletionPolicy: deletionPolicy,
	}
}

func (h *Handler) Name() string { return "asset" }

// Activate installs the account-integrity guard only when asset is selected.
func (h *Handler) Activate() error {
	return h.deletionPolicy.Register(h.deletionGuard)
}

func (h *Handler) List(c *gin.Context) {
	setPrivateNoStore(c)
	userID, ok := httphandler.GetUserID(c)
	if !ok {
		return
	}
	status := c.DefaultQuery("status", "all")
	page := pagination.FromContext(c)
	items, total, err := h.service.ListForUser(
		c.Request.Context(),
		userID,
		status,
		page.GetPage(),
		page.GetPerPage(),
	)
	if err != nil {
		response.HandleError(c, "Failed to list assets", err)
		return
	}
	values := make([]*AssetResponse, len(items))
	for index := range items {
		values[index] = toAssetResponse(items[index])
	}
	paginator := pagination.NewPaginator(values, total, page.GetPage(), page.GetPerPage())
	paginator.SetPath(c.Request.URL.Path)
	paginator.Append("status", status)
	response.Success(c, paginator)
}

func (h *Handler) CreateUploadIntent(c *gin.Context) {
	setPrivateNoStore(c)
	userID, ok := httphandler.GetUserID(c)
	if !ok {
		return
	}
	var request createUploadIntentRequest
	if !httphandler.BindJSON(c, &request) {
		return
	}
	intent, err := h.service.CreateUploadIntent(
		c.Request.Context(),
		userID,
		request.IdempotencyKey,
		request.OriginalName,
		request.MediaType,
		request.SizeBytes,
	)
	if err != nil {
		response.HandleError(c, "Failed to create asset upload intent", err)
		return
	}
	response.Created(c, &UploadIntentResponse{
		Asset:  toAssetResponse(intent.Asset),
		Upload: toTransferGrantResponse(intent.Upload),
	})
}

func (h *Handler) Complete(c *gin.Context) {
	setPrivateNoStore(c)
	userID, ok := httphandler.GetUserID(c)
	if !ok {
		return
	}
	asset, err := h.service.Complete(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		response.HandleError(c, "Failed to complete asset upload", err)
		return
	}
	response.Success(c, toAssetResponse(asset))
}

func (h *Handler) DownloadGrant(c *gin.Context) {
	setPrivateNoStore(c)
	userID, ok := httphandler.GetUserID(c)
	if !ok {
		return
	}
	grant, err := h.service.DownloadGrant(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		response.HandleError(c, "Failed to create asset download grant", err)
		return
	}
	response.Success(c, toTransferGrantResponse(grant))
}

func (h *Handler) Delete(c *gin.Context) {
	setPrivateNoStore(c)
	userID, ok := httphandler.GetUserID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), userID, c.Param("id")); err != nil {
		response.HandleError(c, "Failed to delete asset", err)
		return
	}
	response.NoContent(c)
}

func (h *Handler) LocalUpload(c *gin.Context) {
	setPrivateNoStore(c)
	if err := h.service.AcceptLocalUpload(
		c.Request.Context(),
		c.Param("token"),
		c.GetHeader("Content-Type"),
		c.Request.ContentLength,
		c.Request.Body,
	); err != nil {
		response.HandleError(c, "Asset upload failed", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) LocalDownload(c *gin.Context) {
	setPrivateNoStore(c)
	download, err := h.service.OpenLocalDownload(c.Request.Context(), c.Param("token"))
	if err != nil {
		response.HandleError(c, "Asset download failed", err)
		return
	}
	defer download.Body.Close()
	disposition := mime.FormatMediaType("attachment", map[string]string{
		"filename": download.Asset.OriginalName,
	})
	if disposition == "" {
		response.InternalServerError(c, "Asset download failed", nil)
		return
	}
	c.Header("Content-Disposition", disposition)
	c.DataFromReader(
		http.StatusOK,
		download.Asset.SizeBytes,
		download.Asset.MediaType,
		download.Body,
		nil,
	)
}

func setPrivateNoStore(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	c.Header("Pragma", "no-cache")
}
