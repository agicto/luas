package asset

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storagecap "github.com/zgiai/luas/api/internal/capabilities/storage"
	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/infra/database"
	infrastorage "github.com/zgiai/luas/api/internal/infra/storage"
	"github.com/zgiai/luas/api/internal/modules/user"
)

func newAssetServiceTest(t *testing.T) (*service, uint) {
	t.Helper()
	db, err := database.NewTestDB()
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&user.UserPO{}, &AssetPO{}))
	owner := &user.UserPO{
		Username: "asset-owner",
		Email:    "asset-owner@example.test",
		Password: "hashed-password",
		Status:   1,
	}
	require.NoError(t, db.Create(owner).Error)

	objects, err := infrastorage.NewLocalStore(t.TempDir())
	require.NoError(t, err)
	cfg := &config.Config{
		App:      config.AppConfig{URL: "http://127.0.0.1:8025"},
		Starters: config.StarterConfig{Optional: []string{"asset"}},
		JWT:      config.JWTConfig{Secret: strings.Repeat("s", 64)},
		Asset: config.AssetConfig{
			MaxBytes:         10 * 1024 * 1024,
			UploadGrantTTL:   10 * time.Minute,
			DownloadGrantTTL: 5 * time.Minute,
			PendingTTL:       time.Hour,
		},
	}
	signer, err := NewTransferSigner(cfg)
	require.NoError(t, err)
	fixedNow := time.Date(2026, 7, 15, 20, 0, 0, 0, time.UTC)
	signer.now = func() time.Time { return fixedNow }
	value := NewService(cfg, NewRepository(db), objects, NewContentInspector(), signer)
	value.now = func() time.Time { return fixedNow }
	return value, owner.ID
}

func TestServiceLocalAssetLifecycle(t *testing.T) {
	service, ownerID := newAssetServiceTest(t)
	ctx := context.Background()
	content := []byte("%PDF-1.7\nprivate report\n%%EOF\n")

	intent, err := service.CreateUploadIntent(
		ctx,
		ownerID,
		"asset-lifecycle-1",
		"quarterly-report.pdf",
		"application/pdf",
		int64(len(content)),
	)
	require.NoError(t, err)
	require.NotNil(t, intent)
	assert.Equal(t, domain.AssetStatusPending, intent.Asset.Status)
	assert.Equal(t, "PUT", intent.Upload.Method)
	assert.NotContains(t, intent.Upload.URL, intent.Asset.StagingKey)

	replayed, err := service.CreateUploadIntent(
		ctx,
		ownerID,
		"asset-lifecycle-1",
		"quarterly-report.pdf",
		"application/pdf",
		int64(len(content)),
	)
	require.NoError(t, err)
	assert.Equal(t, intent.Asset.ID, replayed.Asset.ID)

	_, err = service.CreateUploadIntent(
		ctx,
		ownerID,
		"asset-lifecycle-1",
		"different.pdf",
		"application/pdf",
		int64(len(content)),
	)
	assert.ErrorIs(t, err, domain.ErrAssetIdempotencyConflict)

	uploadToken := transferTokenFromURL(t, intent.Upload.URL)
	require.NoError(t, service.AcceptLocalUpload(
		ctx,
		uploadToken,
		"application/pdf",
		int64(len(content)),
		bytes.NewReader(content),
	))

	ready, err := service.Complete(ctx, ownerID, intent.Asset.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.AssetStatusReady, ready.Status)
	assert.NotEmpty(t, ready.ChecksumSHA256)
	assert.NotEmpty(t, ready.StagingKey)

	replayedComplete, err := service.Complete(ctx, ownerID, intent.Asset.ID)
	require.NoError(t, err)
	assert.Equal(t, ready.ID, replayedComplete.ID)
	assert.ErrorIs(t, service.AcceptLocalUpload(
		ctx,
		uploadToken,
		"application/pdf",
		int64(len(content)),
		bytes.NewReader(content),
	), domain.ErrAssetNotReady)

	grant, err := service.DownloadGrant(ctx, ownerID, ready.ID)
	require.NoError(t, err)
	assert.Equal(t, "GET", grant.Method)
	download, err := service.OpenLocalDownload(ctx, transferTokenFromURL(t, grant.URL))
	require.NoError(t, err)
	defer download.Body.Close()
	got, err := io.ReadAll(download.Body)
	require.NoError(t, err)
	assert.Equal(t, content, got)

	ref, err := service.ReadyForUser(ctx, ownerID, ready.ID)
	require.NoError(t, err)
	assert.Equal(t, ready.ID, ref.ID)
	_, err = service.ReadyForUser(ctx, ownerID+1, ready.ID)
	assert.ErrorIs(t, err, domain.ErrAssetNotFound)
	assert.ErrorIs(t, service.CheckAccountDeletion(ctx, ownerID), domain.ErrAssetCleanupRequired)

	require.NoError(t, service.Delete(ctx, ownerID, ready.ID))
	require.NoError(t, service.Delete(ctx, ownerID, ready.ID))
	require.NoError(t, service.CheckAccountDeletion(ctx, ownerID))
	_, err = service.DownloadGrant(ctx, ownerID, ready.ID)
	assert.ErrorIs(t, err, domain.ErrAssetNotFound)
}

func TestServiceRejectsMismatchedAndInvalidContent(t *testing.T) {
	service, ownerID := newAssetServiceTest(t)
	ctx := context.Background()
	content := []byte("this is not a PDF")
	intent, err := service.CreateUploadIntent(
		ctx,
		ownerID,
		"asset-invalid-content",
		"report.pdf",
		"application/pdf",
		int64(len(content)),
	)
	require.NoError(t, err)
	uploadToken := transferTokenFromURL(t, intent.Upload.URL)
	assert.ErrorIs(t, service.AcceptLocalUpload(
		ctx,
		uploadToken,
		"text/plain",
		int64(len(content)),
		bytes.NewReader(content),
	), domain.ErrAssetInvalidMediaType)
	require.NoError(t, service.AcceptLocalUpload(
		ctx,
		uploadToken,
		"application/pdf",
		int64(len(content)),
		bytes.NewReader(content),
	))
	_, err = service.Complete(ctx, ownerID, intent.Asset.ID)
	assert.ErrorIs(t, err, domain.ErrAssetInvalidMediaType)
	_, err = service.DownloadGrant(ctx, ownerID, intent.Asset.ID)
	assert.ErrorIs(t, err, domain.ErrAssetNotReady)
}

func TestServicePrunesExpiredPendingAsset(t *testing.T) {
	service, ownerID := newAssetServiceTest(t)
	ctx := context.Background()
	content := []byte("hello")
	intent, err := service.CreateUploadIntent(
		ctx,
		ownerID,
		"asset-expired",
		"note.txt",
		"text/plain",
		int64(len(content)),
	)
	require.NoError(t, err)
	require.NoError(t, service.AcceptLocalUpload(
		ctx,
		transferTokenFromURL(t, intent.Upload.URL),
		"text/plain",
		int64(len(content)),
		bytes.NewReader(content),
	))
	service.now = func() time.Time { return intent.Asset.PendingExpiresAt.Add(time.Second) }
	processed, err := service.Prune(ctx, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)
	_, err = service.Complete(ctx, ownerID, intent.Asset.ID)
	assert.True(t, errors.Is(err, domain.ErrAssetNotReady) || errors.Is(err, domain.ErrAssetUploadExpired))
}

func TestServiceInspectsFrozenFinalObjectAfterStagingChanges(t *testing.T) {
	service, ownerID := newAssetServiceTest(t)
	ctx := context.Background()
	content := []byte("%PDF-1.7\ntrusted content\n")
	service.objects = &overwriteBeforeCopyStore{
		ObjectStore: service.objects,
		replacement: bytes.Repeat([]byte{'x'}, len(content)),
	}

	intent, err := service.CreateUploadIntent(
		ctx,
		ownerID,
		"asset-staging-race",
		"report.pdf",
		"application/pdf",
		int64(len(content)),
	)
	require.NoError(t, err)
	require.NoError(t, service.AcceptLocalUpload(
		ctx,
		transferTokenFromURL(t, intent.Upload.URL),
		"application/pdf",
		int64(len(content)),
		bytes.NewReader(content),
	))

	_, err = service.Complete(ctx, ownerID, intent.Asset.ID)
	assert.ErrorIs(t, err, domain.ErrAssetInvalidMediaType)
	_, err = service.DownloadGrant(ctx, ownerID, intent.Asset.ID)
	assert.ErrorIs(t, err, domain.ErrAssetNotReady)
}

func TestServicePrunesLateReadyStagingWithoutDeletingFinalObject(t *testing.T) {
	service, ownerID := newAssetServiceTest(t)
	ctx := context.Background()
	content := []byte("ready object")
	intent, err := service.CreateUploadIntent(
		ctx,
		ownerID,
		"asset-ready-staging-cleanup",
		"ready.txt",
		"text/plain",
		int64(len(content)),
	)
	require.NoError(t, err)
	require.NoError(t, service.AcceptLocalUpload(
		ctx,
		transferTokenFromURL(t, intent.Upload.URL),
		"text/plain",
		int64(len(content)),
		bytes.NewReader(content),
	))
	ready, err := service.Complete(ctx, ownerID, intent.Asset.ID)
	require.NoError(t, err)
	require.NotEmpty(t, ready.StagingKey)
	require.NoError(t, service.objects.Put(
		ctx,
		ready.StagingKey,
		bytes.NewReader(content),
		int64(len(content)),
		"text/plain",
	))

	service.now = func() time.Time { return ready.PendingExpiresAt.Add(time.Second) }
	processed, err := service.Prune(ctx, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)

	stored, err := service.store.FindForUser(ctx, ownerID, ready.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.AssetStatusReady, stored.Status)
	assert.Empty(t, stored.StagingKey)
	assert.NotEmpty(t, stored.ObjectKey)
	reader, err := service.objects.Open(ctx, stored.ObjectKey)
	require.NoError(t, err)
	defer reader.Close()
	got, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, content, got)
}

func TestServicePrunesLateDeletedStagingAfterGrantExpiry(t *testing.T) {
	service, ownerID := newAssetServiceTest(t)
	ctx := context.Background()
	content := []byte("late provider upload")
	intent, err := service.CreateUploadIntent(
		ctx,
		ownerID,
		"asset-deleted-staging-cleanup",
		"late.txt",
		"text/plain",
		int64(len(content)),
	)
	require.NoError(t, err)

	require.NoError(t, service.Delete(ctx, ownerID, intent.Asset.ID))
	deleted, err := service.store.FindForUser(ctx, ownerID, intent.Asset.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.AssetStatusDeleted, deleted.Status)
	assert.NotEmpty(t, deleted.StagingKey)
	assert.NotEmpty(t, deleted.ObjectKey)

	// A provider-native PUT grant cannot be revoked. Recreate the late staging write
	// directly, then prove expiry cleanup can still locate and remove it.
	require.NoError(t, service.objects.Put(
		ctx,
		deleted.StagingKey,
		bytes.NewReader(content),
		int64(len(content)),
		"text/plain",
	))
	service.now = func() time.Time { return deleted.PendingExpiresAt.Add(time.Second) }
	processed, err := service.Prune(ctx, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)

	cleaned, err := service.store.FindForUser(ctx, ownerID, deleted.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.AssetStatusDeleted, cleaned.Status)
	assert.NotNil(t, cleaned.DeletedAt)
	assert.Empty(t, cleaned.StagingKey)
	assert.Empty(t, cleaned.ObjectKey)
	_, err = service.objects.Open(ctx, deleted.StagingKey)
	assert.ErrorIs(t, err, storagecap.ErrObjectNotFound)
}

type overwriteBeforeCopyStore struct {
	storagecap.ObjectStore
	replacement []byte
}

func (s *overwriteBeforeCopyStore) Copy(ctx context.Context, sourceKey, destinationKey string) error {
	if err := s.Put(
		ctx,
		sourceKey,
		bytes.NewReader(s.replacement),
		int64(len(s.replacement)),
		"application/pdf",
	); err != nil {
		return err
	}
	return s.ObjectStore.Copy(ctx, sourceKey, destinationKey)
}

func transferTokenFromURL(t *testing.T, value string) string {
	t.Helper()
	parsed, err := url.Parse(value)
	require.NoError(t, err)
	return path.Base(parsed.Path)
}
