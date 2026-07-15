package storage

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"

	storagecap "github.com/zgiai/luas/api/internal/capabilities/storage"
)

type fakeS3Client struct{}

func (fakeS3Client) PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return &s3.PutObjectOutput{}, nil
}
func (fakeS3Client) GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return nil, &smithy.GenericAPIError{Code: "NoSuchKey", Message: "missing"}
}
func (fakeS3Client) HeadObject(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	return nil, &smithy.GenericAPIError{Code: "NotFound", Message: "missing"}
}
func (fakeS3Client) CopyObject(context.Context, *s3.CopyObjectInput, ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
	return &s3.CopyObjectOutput{}, nil
}
func (fakeS3Client) DeleteObject(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	return &s3.DeleteObjectOutput{}, nil
}

type fakePresigner struct {
	putInput *s3.PutObjectInput
	getInput *s3.GetObjectInput
}

func (f *fakePresigner) PresignPutObject(_ context.Context, input *s3.PutObjectInput, _ ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
	f.putInput = input
	return &v4.PresignedHTTPRequest{
		Method: http.MethodPut,
		URL:    "https://account.r2.cloudflarestorage.com/private/asset?X-Amz-Signature=secret",
		SignedHeader: http.Header{
			"Content-Type": []string{"application/pdf"},
			"Host":         []string{"account.r2.cloudflarestorage.com"},
		},
	}, nil
}

func (f *fakePresigner) PresignGetObject(_ context.Context, input *s3.GetObjectInput, _ ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
	f.getInput = input
	return &v4.PresignedHTTPRequest{
		Method: http.MethodGet,
		URL:    "https://account.r2.cloudflarestorage.com/private/asset?X-Amz-Signature=secret",
	}, nil
}

func TestR2StorePresignsBoundedUploadAndAttachmentDownload(t *testing.T) {
	presigner := &fakePresigner{}
	now := time.Date(2026, 7, 15, 20, 0, 0, 0, time.UTC)
	store := &R2Store{
		client:    fakeS3Client{},
		presigner: presigner,
		bucket:    "private",
		now:       func() time.Time { return now },
	}

	upload, err := store.PresignUpload(context.Background(), storagecap.UploadGrantOptions{
		Key:       "asset-uploads/asset-id/object",
		MediaType: "application/pdf",
		TTL:       10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("PresignUpload() error = %v", err)
	}
	if upload.Method != http.MethodPut || upload.Headers["content-type"] != "application/pdf" {
		t.Fatalf("upload grant = %#v", upload)
	}
	if _, exists := upload.Headers["host"]; exists {
		t.Fatalf("upload grant exposed browser-forbidden Host header: %#v", upload.Headers)
	}
	if upload.ExpiresAt != now.Add(10*time.Minute) {
		t.Fatalf("upload expiry = %s", upload.ExpiresAt)
	}
	if got := *presigner.putInput.Key; got != "asset-uploads/asset-id/object" {
		t.Fatalf("upload key = %q", got)
	}

	download, err := store.PresignDownload(context.Background(), storagecap.DownloadGrantOptions{
		Key:          "assets/asset-id/object",
		MediaType:    "application/pdf",
		DownloadName: "quarterly report.pdf",
		TTL:          5 * time.Minute,
	})
	if err != nil {
		t.Fatalf("PresignDownload() error = %v", err)
	}
	if download.Method != http.MethodGet || download.ExpiresAt != now.Add(5*time.Minute) {
		t.Fatalf("download grant = %#v", download)
	}
	disposition := *presigner.getInput.ResponseContentDisposition
	if !strings.HasPrefix(disposition, "attachment;") || !strings.Contains(disposition, "quarterly report.pdf") {
		t.Fatalf("download disposition = %q", disposition)
	}
	parsed, err := url.Parse(download.URL)
	if err != nil || parsed.Query().Get("X-Amz-Signature") == "" {
		t.Fatalf("download URL = %q, error = %v", download.URL, err)
	}
}

func TestR2StoreMapsMissingObjectsWithoutProviderDetail(t *testing.T) {
	store := &R2Store{client: fakeS3Client{}, bucket: "private"}
	_, err := store.Stat(context.Background(), "assets/missing/object")
	if !errors.Is(err, storagecap.ErrObjectNotFound) {
		t.Fatalf("Stat() error = %v, want ErrObjectNotFound", err)
	}
	if strings.Contains(err.Error(), "missing") {
		t.Fatalf("mapped error leaked provider detail: %v", err)
	}
}

func TestR2StoreRejectsUnsafeGrantInputs(t *testing.T) {
	store := &R2Store{
		client:    fakeS3Client{},
		presigner: &fakePresigner{},
		bucket:    "private",
		now:       time.Now,
	}
	tests := []storagecap.DownloadGrantOptions{
		{Key: "../escape", MediaType: "text/plain", DownloadName: "safe.txt", TTL: time.Minute},
		{Key: "assets/safe/object", MediaType: "text/plain", DownloadName: "../escape.txt", TTL: time.Minute},
		{Key: "assets/safe/object", MediaType: "text/plain", DownloadName: "safe.txt\r\nX-Evil: yes", TTL: time.Minute},
		{Key: "assets/safe/object", MediaType: "text/plain", DownloadName: "safe.txt", TTL: 2 * time.Hour},
	}
	for _, input := range tests {
		if _, err := store.PresignDownload(context.Background(), input); err == nil {
			t.Fatalf("PresignDownload(%#v) error = nil, want rejection", input)
		}
	}
}
