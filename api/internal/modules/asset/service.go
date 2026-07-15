package asset

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	storagecap "github.com/zgiai/luas/api/internal/capabilities/storage"
	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
	auditstarter "github.com/zgiai/luas/api/internal/modules/audit"
	"github.com/zgiai/luas/api/internal/modules/user"
)

const (
	assetOperationLease = 2 * time.Minute
	maxAssetPruneBatch  = 100
)

type uploadIntent struct {
	Asset  *domain.Asset
	Upload storagecap.TransferGrant
}

type localDownload struct {
	Asset *domain.Asset
	Body  io.ReadCloser
}

// Service owns user asset metadata, object promotion, and private transfer grants.
type Service interface {
	domain.AssetReader
	domain.AssetMaintainer
	user.AccountDeletionGuard
	CreateUploadIntent(context.Context, uint, string, string, string, int64) (*uploadIntent, error)
	ListForUser(context.Context, uint, string, int, int) ([]*domain.Asset, int64, error)
	Complete(context.Context, uint, string) (*domain.Asset, error)
	DownloadGrant(context.Context, uint, string) (storagecap.TransferGrant, error)
	Delete(context.Context, uint, string) error
	AcceptLocalUpload(context.Context, string, string, int64, io.Reader) error
	OpenLocalDownload(context.Context, string) (*localDownload, error)
}

type service struct {
	enabled          bool
	appURL           string
	maxBytes         int64
	uploadGrantTTL   time.Duration
	downloadGrantTTL time.Duration
	pendingTTL       time.Duration
	store            assetStore
	objects          storagecap.ObjectStore
	inspector        contentInspector
	transferSigner   *transferSigner
	now              func() time.Time
}

var (
	_ Service                   = (*service)(nil)
	_ domain.AssetReader        = (*service)(nil)
	_ domain.AssetMaintainer    = (*service)(nil)
	_ user.AccountDeletionGuard = (*service)(nil)
)

// NewContentInspector creates the conservative default content-integrity inspector.
func NewContentInspector() contentInspector { return newBaselineInspector() }

// NewTransferSigner creates the local development transfer signer from the auth secret.
func NewTransferSigner(cfg *config.Config) (*transferSigner, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required for asset transfer signing")
	}
	if !slices.Contains(cfg.Starters.Optional, "asset") {
		return &transferSigner{}, nil
	}
	return newTransferSigner(cfg.JWT.Secret)
}

// NewService creates the optional private asset service.
func NewService(
	cfg *config.Config,
	store assetStore,
	objects storagecap.ObjectStore,
	inspector contentInspector,
	signer *transferSigner,
) *service {
	value := &service{store: store, objects: objects, inspector: inspector, transferSigner: signer}
	if cfg != nil {
		value.enabled = slices.Contains(cfg.Starters.Optional, "asset")
		value.appURL = cfg.App.URL
		value.maxBytes = cfg.Asset.MaxBytes
		value.uploadGrantTTL = cfg.Asset.UploadGrantTTL
		value.downloadGrantTTL = cfg.Asset.DownloadGrantTTL
		value.pendingTTL = cfg.Asset.PendingTTL
	}
	value.now = func() time.Time { return time.Now().UTC() }
	return value
}

func (s *service) CreateUploadIntent(
	ctx context.Context,
	userID uint,
	idempotencyKey string,
	originalName string,
	mediaType string,
	sizeBytes int64,
) (*uploadIntent, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	if userID == 0 {
		return nil, domain.ErrInvalidInput
	}
	idempotencyKey, originalName, mediaType, err := normalizeUploadMetadata(
		idempotencyKey,
		originalName,
		mediaType,
		sizeBytes,
		s.maxBytes,
	)
	if err != nil {
		return nil, err
	}
	requestHash, err := assetRequestFingerprint(originalName, mediaType, sizeBytes)
	if err != nil {
		return nil, domain.ErrServiceUnavailable
	}
	now := s.now()
	id := uuid.NewString()
	result, err := s.store.CreateIntent(ctx, &domain.Asset{
		ID:               id,
		UserID:           userID,
		IdempotencyKey:   idempotencyKey,
		RequestHash:      requestHash,
		OriginalName:     originalName,
		MediaType:        mediaType,
		SizeBytes:        sizeBytes,
		Status:           domain.AssetStatusPending,
		StagingKey:       "asset-uploads/" + id + "/object",
		ObjectKey:        "assets/" + id + "/object",
		PendingExpiresAt: now.Add(s.pendingTTL),
		CreatedAt:        now,
		UpdatedAt:        now,
	})
	if err != nil {
		return nil, fmt.Errorf("create asset upload intent: %w", err)
	}
	if result == nil || result.Asset == nil {
		return nil, domain.ErrServiceUnavailable
	}
	asset := result.Asset
	if asset.Status != domain.AssetStatusPending {
		return nil, domain.ErrAssetNotReady
	}
	if !asset.PendingExpiresAt.After(now) {
		return nil, domain.ErrAssetUploadExpired
	}
	grant, err := s.uploadGrant(ctx, asset)
	if err != nil {
		return nil, err
	}
	if result.Created {
		recordAssetAudit(ctx, "create", asset, "")
	}
	return &uploadIntent{Asset: asset, Upload: grant}, nil
}

func (s *service) ListForUser(
	ctx context.Context,
	userID uint,
	status string,
	page int,
	pageSize int,
) ([]*domain.Asset, int64, error) {
	if err := s.available(); err != nil {
		return nil, 0, err
	}
	if userID == 0 || !validAssetStatusFilter(status) || page < 1 || pageSize < 1 || pageSize > 100 {
		return nil, 0, domain.ErrInvalidInput
	}
	return s.store.ListForUser(ctx, userID, status, page, pageSize)
}

func (s *service) Complete(ctx context.Context, userID uint, id string) (*domain.Asset, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	if userID == 0 || !validAssetID(id) {
		return nil, domain.ErrInvalidInput
	}
	now := s.now()
	operationToken := uuid.NewString()
	claim, err := s.store.ClaimCompletion(ctx, userID, id, operationToken, now, now.Add(assetOperationLease))
	if err != nil {
		return nil, err
	}
	if claim == nil || claim.Asset == nil {
		return nil, domain.ErrServiceUnavailable
	}
	if !claim.Claimed {
		return claim.Asset, nil
	}
	release := true
	defer func() {
		if release {
			releaseAssetOperationBestEffort(ctx, s.store, id, operationToken, s.now())
		}
	}()

	asset := claim.Asset
	sourceKey := asset.StagingKey
	info, statErr := s.objects.Stat(ctx, sourceKey)
	if errors.Is(statErr, storagecap.ErrObjectNotFound) {
		sourceKey = asset.ObjectKey
		info, statErr = s.objects.Stat(ctx, sourceKey)
	}
	if statErr != nil {
		if errors.Is(statErr, storagecap.ErrObjectNotFound) {
			rejected, rejectErr := s.reject(ctx, asset, operationToken, domain.CodeAssetNotReady, domain.ErrAssetNotReady)
			release = rejectErr != nil
			return rejected, rejectErr
		}
		return nil, mapObjectError(statErr)
	}
	if info.Size != asset.SizeBytes || info.Size > s.maxBytes {
		rejected, rejectErr := s.reject(ctx, asset, operationToken, domain.CodeAssetSizeExceeded, domain.ErrAssetSizeExceeded)
		release = rejectErr != nil
		return rejected, rejectErr
	}
	if info.MediaType != "" && info.MediaType != asset.MediaType {
		rejected, rejectErr := s.reject(ctx, asset, operationToken, domain.CodeAssetInvalidMediaType, domain.ErrAssetInvalidMediaType)
		release = rejectErr != nil
		return rejected, rejectErr
	}

	// Freeze staging bytes under the immutable final key before inspection. A still-valid
	// provider upload grant can replace staging, so inspecting staging and copying later is unsafe.
	if sourceKey != asset.ObjectKey {
		if copyErr := s.objects.Copy(ctx, sourceKey, asset.ObjectKey); copyErr != nil {
			return nil, mapObjectError(copyErr)
		}
		sourceKey = asset.ObjectKey
		info, statErr = s.objects.Stat(ctx, sourceKey)
		if statErr != nil {
			return nil, mapObjectError(statErr)
		}
		if info.Size != asset.SizeBytes || info.Size > s.maxBytes {
			rejected, rejectErr := s.reject(ctx, asset, operationToken, domain.CodeAssetSizeExceeded, domain.ErrAssetSizeExceeded)
			release = rejectErr != nil
			return rejected, rejectErr
		}
		if info.MediaType != "" && info.MediaType != asset.MediaType {
			rejected, rejectErr := s.reject(ctx, asset, operationToken, domain.CodeAssetInvalidMediaType, domain.ErrAssetInvalidMediaType)
			release = rejectErr != nil
			return rejected, rejectErr
		}
	}

	reader, err := s.objects.Open(ctx, asset.ObjectKey)
	if err != nil {
		return nil, mapObjectError(err)
	}
	checksum, inspectErr := s.inspector.Inspect(ctx, reader, asset.SizeBytes, s.maxBytes, asset.MediaType)
	closeErr := reader.Close()
	if inspectErr == nil && closeErr != nil {
		inspectErr = domain.ErrServiceUnavailable
	}
	if inspectErr != nil {
		if errors.Is(inspectErr, domain.ErrAssetInvalidMediaType) || errors.Is(inspectErr, domain.ErrAssetSizeExceeded) {
			code := domain.CodeAssetInvalidMediaType
			if errors.Is(inspectErr, domain.ErrAssetSizeExceeded) {
				code = domain.CodeAssetSizeExceeded
			}
			rejected, rejectErr := s.reject(ctx, asset, operationToken, code, inspectErr)
			release = rejectErr != nil
			return rejected, rejectErr
		}
		return nil, inspectErr
	}

	deleteAssetObjectBestEffort(ctx, s.objects, asset.StagingKey)
	ready, err := s.store.MarkReady(ctx, id, operationToken, checksum, s.now())
	if err != nil {
		return nil, err
	}
	release = false
	recordAssetAudit(ctx, "update", ready, "")
	return ready, nil
}

func (s *service) DownloadGrant(
	ctx context.Context,
	userID uint,
	id string,
) (storagecap.TransferGrant, error) {
	if err := s.available(); err != nil {
		return storagecap.TransferGrant{}, err
	}
	if userID == 0 || !validAssetID(id) {
		return storagecap.TransferGrant{}, domain.ErrInvalidInput
	}
	asset, err := s.store.FindForUser(ctx, userID, id)
	if err != nil {
		return storagecap.TransferGrant{}, err
	}
	if asset.DeletedAt != nil || asset.Status == domain.AssetStatusDeleted {
		return storagecap.TransferGrant{}, domain.ErrAssetNotFound
	}
	if asset.Status != domain.AssetStatusReady {
		return storagecap.TransferGrant{}, domain.ErrAssetNotReady
	}
	return s.downloadGrant(ctx, asset)
}

func (s *service) Delete(ctx context.Context, userID uint, id string) error {
	if err := s.available(); err != nil {
		return err
	}
	if userID == 0 || !validAssetID(id) {
		return domain.ErrInvalidInput
	}
	now := s.now()
	operationToken := uuid.NewString()
	claim, err := s.store.ClaimDeletion(ctx, userID, id, operationToken, now, now.Add(assetOperationLease))
	if err != nil {
		return err
	}
	if claim == nil || claim.Asset == nil {
		return domain.ErrServiceUnavailable
	}
	if !claim.Claimed {
		return nil
	}
	release := true
	defer func() {
		if release {
			releaseAssetOperationBestEffort(ctx, s.store, id, operationToken, s.now())
		}
	}()
	if deleteErr := s.deleteAssetObjects(ctx, claim.Asset); deleteErr != nil {
		return deleteErr
	}
	deleted, err := s.store.MarkDeleted(ctx, id, operationToken, s.now())
	if err != nil {
		return err
	}
	release = false
	recordAssetAudit(ctx, "delete", deleted, "")
	return nil
}

func (s *service) ReadyForUser(
	ctx context.Context,
	userID uint,
	assetID string,
) (domain.AssetReference, error) {
	if err := s.available(); err != nil {
		return domain.AssetReference{}, err
	}
	if userID == 0 || !validAssetID(assetID) {
		return domain.AssetReference{}, domain.ErrInvalidInput
	}
	asset, err := s.store.FindForUser(ctx, userID, assetID)
	if err != nil {
		return domain.AssetReference{}, err
	}
	if asset.DeletedAt != nil || asset.Status == domain.AssetStatusDeleted {
		return domain.AssetReference{}, domain.ErrAssetNotFound
	}
	if asset.Status != domain.AssetStatusReady {
		return domain.AssetReference{}, domain.ErrAssetNotReady
	}
	return domain.AssetReference{ID: asset.ID, MediaType: asset.MediaType, SizeBytes: asset.SizeBytes}, nil
}

func (s *service) AcceptLocalUpload(
	ctx context.Context,
	token string,
	mediaType string,
	contentLength int64,
	body io.Reader,
) error {
	if err := s.available(); err != nil {
		return err
	}
	if s.objects.Driver() != storagecap.DriverLocal || s.transferSigner == nil {
		return domain.ErrAssetNotFound
	}
	claims, err := s.transferSigner.Verify(token, "upload")
	if err != nil {
		return domain.ErrAssetNotFound
	}
	asset, err := s.store.FindByID(ctx, claims.AssetID)
	if err != nil {
		return err
	}
	if asset.Status != domain.AssetStatusPending || !asset.PendingExpiresAt.After(s.now()) {
		return domain.ErrAssetNotReady
	}
	if mediaType != asset.MediaType {
		return domain.ErrAssetInvalidMediaType
	}
	if contentLength != asset.SizeBytes {
		return domain.ErrAssetSizeExceeded
	}
	return mapObjectError(s.objects.Put(ctx, asset.StagingKey, body, asset.SizeBytes, asset.MediaType))
}

func (s *service) OpenLocalDownload(ctx context.Context, token string) (*localDownload, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	if s.objects.Driver() != storagecap.DriverLocal || s.transferSigner == nil {
		return nil, domain.ErrAssetNotFound
	}
	claims, err := s.transferSigner.Verify(token, "download")
	if err != nil {
		return nil, domain.ErrAssetNotFound
	}
	asset, err := s.store.FindByID(ctx, claims.AssetID)
	if err != nil {
		return nil, err
	}
	if asset.DeletedAt != nil || asset.Status == domain.AssetStatusDeleted {
		return nil, domain.ErrAssetNotFound
	}
	if asset.Status != domain.AssetStatusReady {
		return nil, domain.ErrAssetNotReady
	}
	body, err := s.objects.Open(ctx, asset.ObjectKey)
	if err != nil {
		return nil, mapObjectError(err)
	}
	return &localDownload{Asset: asset, Body: body}, nil
}

func (s *service) Prune(ctx context.Context, limit int) (int, error) {
	if err := s.available(); err != nil {
		return 0, err
	}
	if limit < 1 || limit > maxAssetPruneBatch {
		return 0, domain.ErrInvalidInput
	}
	now := s.now()
	candidates, err := s.store.ListCleanupCandidates(ctx, now, limit)
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return processed, err
		}
		token := uuid.NewString()
		claim, err := s.store.ClaimPrune(ctx, candidate.ID, token, now, now.Add(assetOperationLease))
		if errors.Is(err, domain.ErrAssetNotReady) {
			continue
		}
		if err != nil {
			return processed, err
		}
		if claim == nil || !claim.Claimed || claim.Asset == nil {
			continue
		}
		preserveFinal := claim.Asset.Status == domain.AssetStatusReady
		var cleanupErr error
		if preserveFinal {
			cleanupErr = mapObjectError(s.objects.Delete(ctx, claim.Asset.StagingKey))
		} else {
			cleanupErr = s.deleteAssetObjects(ctx, claim.Asset)
		}
		if cleanupErr != nil {
			releaseAssetOperationBestEffort(ctx, s.store, claim.Asset.ID, token, s.now())
			return processed, cleanupErr
		}
		if err := s.store.FinishPrune(ctx, claim.Asset.ID, token, domain.CodeAssetUploadExpired, preserveFinal, s.now()); err != nil {
			releaseAssetOperationBestEffort(ctx, s.store, claim.Asset.ID, token, s.now())
			return processed, err
		}
		processed++
	}
	return processed, nil
}

func (s *service) AccountDeletionGuardName() string { return "asset" }

func (s *service) CheckAccountDeletion(ctx context.Context, userID uint) error {
	if err := s.available(); err != nil {
		return err
	}
	if userID == 0 {
		return domain.ErrInvalidInput
	}
	count, err := s.store.CountActiveForUser(ctx, userID)
	if err != nil {
		return err
	}
	if count > 0 {
		return domain.ErrAssetCleanupRequired
	}
	return nil
}

func (s *service) reject(
	ctx context.Context,
	asset *domain.Asset,
	operationToken string,
	reasonCode string,
	publicError error,
) (*domain.Asset, error) {
	if asset.StagingKey != "" {
		deleteAssetObjectBestEffort(ctx, s.objects, asset.StagingKey)
	}
	if asset.ObjectKey != "" {
		deleteAssetObjectBestEffort(ctx, s.objects, asset.ObjectKey)
	}
	rejected, err := s.store.MarkRejected(ctx, asset.ID, operationToken, reasonCode, s.now())
	if err != nil {
		return nil, err
	}
	recordAssetAudit(ctx, "update", rejected, reasonCode)
	return rejected, publicError
}

func (s *service) uploadGrant(ctx context.Context, asset *domain.Asset) (storagecap.TransferGrant, error) {
	if direct, ok := s.objects.(storagecap.DirectTransferStore); ok {
		grant, err := direct.PresignUpload(ctx, storagecap.UploadGrantOptions{
			Key:       asset.StagingKey,
			MediaType: asset.MediaType,
			TTL:       s.uploadGrantTTL,
		})
		if err != nil {
			return storagecap.TransferGrant{}, mapObjectError(err)
		}
		return grant, nil
	}
	if s.objects.Driver() != storagecap.DriverLocal || s.transferSigner == nil {
		return storagecap.TransferGrant{}, domain.ErrServiceUnavailable
	}
	token, expires, err := s.transferSigner.Sign("upload", asset.ID, s.uploadGrantTTL)
	if err != nil {
		return storagecap.TransferGrant{}, domain.ErrServiceUnavailable
	}
	transferURL, err := localTransferURL(s.appURL, token)
	if err != nil {
		return storagecap.TransferGrant{}, domain.ErrServiceUnavailable
	}
	return storagecap.TransferGrant{
		Method: http.MethodPut,
		URL:    transferURL,
		Headers: map[string]string{
			"content-type": asset.MediaType,
		},
		ExpiresAt: expires,
	}, nil
}

func (s *service) downloadGrant(ctx context.Context, asset *domain.Asset) (storagecap.TransferGrant, error) {
	if direct, ok := s.objects.(storagecap.DirectTransferStore); ok {
		grant, err := direct.PresignDownload(ctx, storagecap.DownloadGrantOptions{
			Key:          asset.ObjectKey,
			MediaType:    asset.MediaType,
			DownloadName: asset.OriginalName,
			TTL:          s.downloadGrantTTL,
		})
		if err != nil {
			return storagecap.TransferGrant{}, mapObjectError(err)
		}
		return grant, nil
	}
	if s.objects.Driver() != storagecap.DriverLocal || s.transferSigner == nil {
		return storagecap.TransferGrant{}, domain.ErrServiceUnavailable
	}
	token, expires, err := s.transferSigner.Sign("download", asset.ID, s.downloadGrantTTL)
	if err != nil {
		return storagecap.TransferGrant{}, domain.ErrServiceUnavailable
	}
	transferURL, err := localTransferURL(s.appURL, token)
	if err != nil {
		return storagecap.TransferGrant{}, domain.ErrServiceUnavailable
	}
	return storagecap.TransferGrant{
		Method:    http.MethodGet,
		URL:       transferURL,
		Headers:   map[string]string{},
		ExpiresAt: expires,
	}, nil
}

func (s *service) deleteAssetObjects(ctx context.Context, asset *domain.Asset) error {
	if asset == nil {
		return domain.ErrInvalidInput
	}
	for _, key := range []string{asset.StagingKey, asset.ObjectKey} {
		if key == "" {
			continue
		}
		if err := s.objects.Delete(ctx, key); err != nil {
			return mapObjectError(err)
		}
	}
	return nil
}

func releaseAssetOperationBestEffort(
	ctx context.Context,
	store assetStore,
	id string,
	token string,
	now time.Time,
) {
	_ = store.ReleaseOperation(context.WithoutCancel(ctx), id, token, now) //nolint:errcheck // lease expiry remains the fallback
}

func deleteAssetObjectBestEffort(ctx context.Context, objects storagecap.ObjectStore, key string) {
	if key == "" {
		return
	}
	_ = objects.Delete(context.WithoutCancel(ctx), key) //nolint:errcheck // expiry cleanup retries provider deletion
}

func (s *service) available() error {
	if s == nil || !s.enabled || s.store == nil || s.objects == nil || s.inspector == nil || s.now == nil {
		return domain.ErrServiceUnavailable
	}
	return nil
}

func assetRequestFingerprint(originalName string, mediaType string, sizeBytes int64) (string, error) {
	payload, err := json.Marshal(struct {
		OriginalName string `json:"original_name"`
		MediaType    string `json:"media_type"`
		SizeBytes    int64  `json:"size_bytes"`
	}{OriginalName: originalName, MediaType: mediaType, SizeBytes: sizeBytes})
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(payload)
	return hex.EncodeToString(hash[:]), nil
}

func validAssetID(id string) bool {
	parsed, err := uuid.Parse(id)
	return err == nil && parsed.String() == strings.ToLower(id)
}

func localTransferURL(appURL string, token string) (string, error) {
	base, err := url.Parse(strings.TrimSpace(appURL))
	if err != nil || base.Scheme == "" || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return "", fmt.Errorf("valid APP_URL origin is required")
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/v1/asset-transfers/" + token
	base.RawPath = ""
	return base.String(), nil
}

func mapObjectError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	switch {
	case errors.Is(err, storagecap.ErrObjectSizeMismatch):
		return domain.ErrAssetSizeExceeded
	case errors.Is(err, storagecap.ErrObjectNotFound):
		return domain.ErrAssetNotReady
	default:
		return domain.ErrServiceUnavailable
	}
}

func recordAssetAudit(ctx context.Context, action string, asset *domain.Asset, reason string) {
	if asset == nil {
		return
	}
	metadata := map[string]any{
		"status":     asset.Status,
		"media_type": asset.MediaType,
		"size_bytes": asset.SizeBytes,
	}
	if reason != "" {
		metadata["reason_code"] = reason
	}
	auditstarter.RecordChange(ctx, auditstarter.Change{
		Action:     action,
		Resource:   "assets",
		TargetType: "asset",
		TargetID:   asset.ID,
		Result:     domain.AuditResultSuccess,
		Metadata:   metadata,
	})
}
