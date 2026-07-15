package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"

	storagecap "github.com/zgiai/luas/api/internal/capabilities/storage"
)

// R2Options configures the private S3-compatible Cloudflare R2 adapter.
type R2Options struct {
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	Region          string
	Endpoint        string
	RequestTimeout  time.Duration
}

type s3API interface {
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	HeadObject(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	CopyObject(context.Context, *s3.CopyObjectInput, ...func(*s3.Options)) (*s3.CopyObjectOutput, error)
	DeleteObject(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

type s3Presigner interface {
	PresignPutObject(context.Context, *s3.PutObjectInput, ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
	PresignGetObject(context.Context, *s3.GetObjectInput, ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
}

// R2Store implements ObjectStore and provider-native direct transfer grants.
type R2Store struct {
	client    s3API
	presigner s3Presigner
	bucket    string
	now       func() time.Time
}

var (
	_ storagecap.ObjectStore         = (*R2Store)(nil)
	_ storagecap.DirectTransferStore = (*R2Store)(nil)
)

// NewR2Store creates an AWS SDK for Go v2 client without consulting ambient AWS profiles.
func NewR2Store(options R2Options) (*R2Store, error) {
	if strings.TrimSpace(options.AccessKeyID) == "" ||
		strings.TrimSpace(options.SecretAccessKey) == "" ||
		strings.TrimSpace(options.Bucket) == "" ||
		strings.TrimSpace(options.Region) == "" ||
		strings.TrimSpace(options.Endpoint) == "" ||
		options.RequestTimeout <= 0 {
		return nil, fmt.Errorf("complete R2 options are required")
	}
	endpoint, err := url.Parse(options.Endpoint)
	if err != nil || endpoint.Host == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") {
		return nil, fmt.Errorf("valid R2 endpoint is required")
	}

	awsConfig := aws.Config{
		Region:                     options.Region,
		Credentials:                aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(options.AccessKeyID, options.SecretAccessKey, "")),
		HTTPClient:                 &http.Client{Timeout: options.RequestTimeout},
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
		RetryMaxAttempts:           3,
		RetryMode:                  aws.RetryModeStandard,
	}
	client := s3.NewFromConfig(awsConfig, func(value *s3.Options) {
		value.BaseEndpoint = aws.String(strings.TrimRight(options.Endpoint, "/"))
		value.UsePathStyle = true
	})
	return &R2Store{
		client:    client,
		presigner: s3.NewPresignClient(client),
		bucket:    options.Bucket,
		now:       func() time.Time { return time.Now().UTC() },
	}, nil
}

func (s *R2Store) Driver() string { return storagecap.DriverR2 }

func (s *R2Store) Put(
	ctx context.Context,
	key string,
	body io.Reader,
	size int64,
	mediaType string,
) error {
	if err := validateR2Operation(ctx, s, key); err != nil {
		return err
	}
	if body == nil || size < 0 {
		return storagecap.ErrObjectSizeMismatch
	}
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          body,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(mediaType),
	}, s3.WithAPIOptions(v4.SwapComputePayloadSHA256ForUnsignedPayloadMiddleware))
	return mapR2Error(err)
}

func (s *R2Store) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := validateR2Operation(ctx, s, key); err != nil {
		return nil, err
	}
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, mapR2Error(err)
	}
	if result == nil || result.Body == nil {
		return nil, storagecap.ErrObjectStoreUnavailable
	}
	return result.Body, nil
}

func (s *R2Store) Stat(ctx context.Context, key string) (storagecap.ObjectInfo, error) {
	if err := validateR2Operation(ctx, s, key); err != nil {
		return storagecap.ObjectInfo{}, err
	}
	result, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return storagecap.ObjectInfo{}, mapR2Error(err)
	}
	if result == nil || result.ContentLength == nil {
		return storagecap.ObjectInfo{}, storagecap.ErrObjectStoreUnavailable
	}
	return storagecap.ObjectInfo{
		Size:         *result.ContentLength,
		MediaType:    aws.ToString(result.ContentType),
		ETag:         strings.Trim(aws.ToString(result.ETag), `"`),
		LastModified: aws.ToTime(result.LastModified),
	}, nil
}

func (s *R2Store) Copy(ctx context.Context, sourceKey, destinationKey string) error {
	if err := validateR2Operation(ctx, s, sourceKey); err != nil {
		return err
	}
	if err := storagecap.ValidateObjectKey(destinationKey); err != nil {
		return err
	}
	copySource := url.PathEscape(s.bucket + "/" + sourceKey)
	_, err := s.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(s.bucket),
		Key:        aws.String(destinationKey),
		CopySource: aws.String(copySource),
	})
	return mapR2Error(err)
}

func (s *R2Store) Delete(ctx context.Context, key string) error {
	if err := validateR2Operation(ctx, s, key); err != nil {
		return err
	}
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return mapR2Error(err)
}

func (s *R2Store) PresignUpload(
	ctx context.Context,
	options storagecap.UploadGrantOptions,
) (storagecap.TransferGrant, error) {
	if err := validateR2Grant(ctx, s, options.Key, options.TTL); err != nil {
		return storagecap.TransferGrant{}, err
	}
	if strings.TrimSpace(options.MediaType) == "" {
		return storagecap.TransferGrant{}, storagecap.ErrDirectTransferUnsupported
	}
	request, err := s.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(options.Key),
		ContentType: aws.String(options.MediaType),
	}, func(value *s3.PresignOptions) {
		value.Expires = options.TTL
	})
	if err != nil {
		return storagecap.TransferGrant{}, mapR2Error(err)
	}
	return transferGrant(request, s.now().Add(options.TTL)), nil
}

func (s *R2Store) PresignDownload(
	ctx context.Context,
	options storagecap.DownloadGrantOptions,
) (storagecap.TransferGrant, error) {
	if err := validateR2Grant(ctx, s, options.Key, options.TTL); err != nil {
		return storagecap.TransferGrant{}, err
	}
	disposition, err := attachmentDisposition(options.DownloadName)
	if err != nil {
		return storagecap.TransferGrant{}, err
	}
	request, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:                     aws.String(s.bucket),
		Key:                        aws.String(options.Key),
		ResponseContentDisposition: aws.String(disposition),
		ResponseContentType:        aws.String(options.MediaType),
	}, func(value *s3.PresignOptions) {
		value.Expires = options.TTL
	})
	if err != nil {
		return storagecap.TransferGrant{}, mapR2Error(err)
	}
	return transferGrant(request, s.now().Add(options.TTL)), nil
}

func transferGrant(request *v4.PresignedHTTPRequest, expiresAt time.Time) storagecap.TransferGrant {
	headers := make(map[string]string, len(request.SignedHeader))
	for name, values := range request.SignedHeader {
		normalized := strings.ToLower(name)
		if len(values) > 0 && browserMaySetHeader(normalized) {
			headers[normalized] = strings.Join(values, ",")
		}
	}
	return storagecap.TransferGrant{
		Method:    request.Method,
		URL:       request.URL,
		Headers:   headers,
		ExpiresAt: expiresAt,
	}
}

func browserMaySetHeader(name string) bool {
	switch name {
	case "authorization", "content-length", "cookie", "host", "origin", "referer", "set-cookie":
		return false
	default:
		return true
	}
}

func attachmentDisposition(name string) (string, error) {
	if name == "" || name != strings.TrimSpace(name) || strings.ContainsAny(name, "\r\n\x00/\\") {
		return "", storagecap.ErrDirectTransferUnsupported
	}
	return mime.FormatMediaType("attachment", map[string]string{"filename": name}), nil
}

func validateR2Grant(ctx context.Context, store *R2Store, key string, ttl time.Duration) error {
	if err := validateR2Operation(ctx, store, key); err != nil {
		return err
	}
	if ttl <= 0 || ttl > time.Hour || store.presigner == nil || store.now == nil {
		return storagecap.ErrDirectTransferUnsupported
	}
	return nil
}

func validateR2Operation(ctx context.Context, store *R2Store, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if store == nil || store.client == nil || strings.TrimSpace(store.bucket) == "" {
		return storagecap.ErrObjectStoreUnavailable
	}
	return storagecap.ValidateObjectKey(key)
}

func mapR2Error(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var apiError smithy.APIError
	if errors.As(err, &apiError) {
		switch apiError.ErrorCode() {
		case "NoSuchKey", "NotFound", "404":
			return storagecap.ErrObjectNotFound
		}
	}
	return storagecap.ErrObjectStoreUnavailable
}
