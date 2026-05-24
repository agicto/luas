package storage

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/zgiai/zgo/internal/infra/config"
	"github.com/google/uuid"
)

// R2Storage implements storage operations for Cloudflare R2
type R2Storage struct {
	client       *s3.S3
	bucket       string
	publicURL    string
	publicDomain string
}

var r2Storage *R2Storage

// InitR2Storage initializes the R2 storage client
func InitR2Storage(cfg *config.Config) error {
	if cfg.R2.AccessKeyID == "" || cfg.R2.SecretAccessKey == "" || cfg.R2.Endpoint == "" || cfg.R2.Bucket == "" {
		return fmt.Errorf("missing required R2 configuration")
	}

	// Create an AWS session
	sess, err := session.NewSession(&aws.Config{
		Credentials:      credentials.NewStaticCredentials(cfg.R2.AccessKeyID, cfg.R2.SecretAccessKey, ""),
		Endpoint:         aws.String(cfg.R2.Endpoint),
		Region:           aws.String(cfg.R2.Region),
		S3ForcePathStyle: aws.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("failed to create R2 session: %w", err)
	}

	// Create S3 service client
	r2Storage = &R2Storage{
		client:       s3.New(sess),
		bucket:       cfg.R2.Bucket,
		publicURL:    cfg.R2.PublicURL,
		publicDomain: cfg.R2.PublicDomain,
	}

	fmt.Printf("Bucket: %s\n", r2Storage.bucket)
	fmt.Printf("Public URL: %s\n", r2Storage.publicURL)
	fmt.Printf("Public Domain: %s\n", r2Storage.publicDomain)
	fmt.Printf("Endpoint: %s\n", *sess.Config.Endpoint)
	fmt.Printf("Region: %s\n", *sess.Config.Region)

	return nil
}

// GetR2Storage returns the R2 storage instance
func GetR2Storage() *R2Storage {
	return r2Storage
}

// UploadFile uploads a file to R2 storage
func (s *R2Storage) UploadFile(data []byte, fileName string, contentType string) (string, error) {
	// Generate a unique file name if not provided
	if fileName == "" {
		ext := filepath.Ext(fileName)
		if ext == "" {
			// Default to .bin if no extension
			ext = ".bin"
		}
		fileName = uuid.New().String() + ext
	}

	// Prepare the S3 input parameters
	params := &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(fileName),
		Body:          bytes.NewReader(data),
		ContentLength: aws.Int64(int64(len(data))),
		ContentType:   aws.String(contentType),
	}

	// Upload the file
	_, err := s.client.PutObject(params)
	if err != nil {
		return "", fmt.Errorf("failed to upload file to R2: %w", err)
	}

	// Return the file URL
	if s.publicURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimRight(s.publicURL, "/"), url.PathEscape(fileName)), nil
	}

	// Use the S3-compatible URL format if no public URL is configured
	return fmt.Sprintf("https://%s.%s/%s", s.bucket, strings.TrimPrefix(strings.TrimPrefix(s.client.Endpoint, "https://"), "http://"), url.PathEscape(fileName)), nil
}

// GetFileURL returns the public URL for a file
func (s *R2Storage) GetFileURL(fileName string) string {
	// Use configured public domain from .env
	if s.publicDomain != "" {
		return fmt.Sprintf("https://%s/%s", s.publicDomain, fileName)
	}

	// Fallback to public URL if available
	if s.publicURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimRight(s.publicURL, "/"), url.PathEscape(fileName))
	}

	// Fallback to S3-compatible URL format
	return fmt.Sprintf("https://%s.%s/%s", s.bucket, strings.TrimPrefix(strings.TrimPrefix(s.client.Endpoint, "https://"), "http://"), url.PathEscape(fileName))
}

// GetPresignedURL returns a presigned URL for a file
func (s *R2Storage) GetPresignedURL(fileName string, expires time.Duration) (string, error) {
	req, _ := s.client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fileName),
	})

	// Generate the presigned URL
	urlStr, err := req.Presign(expires)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return urlStr, nil
}

// DeleteFile deletes a file from R2 storage
func (s *R2Storage) DeleteFile(fileName string) error {
	_, err := s.client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fileName),
	})

	if err != nil {
		return fmt.Errorf("failed to delete file from R2: %w", err)
	}

	return nil
}

// DownloadFile downloads a file from R2 storage
func (s *R2Storage) DownloadFile(fileName string) ([]byte, error) {
	result, err := s.client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fileName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download file from R2: %w", err)
	}
	defer result.Body.Close()

	return io.ReadAll(result.Body)
}

// R2Client represents an R2 storage client
type R2Client struct {
	cfg *config.Config
}

// NewR2Client creates a new R2 client
func NewR2Client(cfg *config.Config) *R2Client {
	return &R2Client{
		cfg: cfg,
	}
}

// FileExists checks if a file exists in R2
func (c *R2Client) FileExists(key string) (bool, error) {
	// Implementation here
	return false, nil
}

// GeneratePresignedURL generates a presigned URL for uploading a file
func (c *R2Client) GeneratePresignedURL(key string, contentType string) (string, error) {
	// Implementation here
	return "", nil
}
