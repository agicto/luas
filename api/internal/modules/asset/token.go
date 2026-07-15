package asset

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const localTransferTokenVersion = "v1"

var errInvalidTransferToken = errors.New("invalid asset transfer token")

type transferClaims struct {
	Operation string
	AssetID   string
	Expires   time.Time
}

type transferSigner struct {
	key []byte
	now func() time.Time
}

func newTransferSigner(secret string) (*transferSigner, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("JWT secret is required for local asset transfer signing")
	}
	derived := sha256.Sum256([]byte("luas:asset-transfer:v1:" + secret))
	return &transferSigner{
		key: derived[:],
		now: func() time.Time { return time.Now().UTC() },
	}, nil
}

func (s *transferSigner) Sign(operation string, assetID string, ttl time.Duration) (string, time.Time, error) {
	if s == nil || len(s.key) == 0 || s.now == nil || !validTransferOperation(operation) || ttl <= 0 || ttl > time.Hour {
		return "", time.Time{}, errInvalidTransferToken
	}
	if _, err := uuid.Parse(assetID); err != nil {
		return "", time.Time{}, errInvalidTransferToken
	}
	expires := s.now().Add(ttl).UTC().Truncate(time.Second)
	payload := strings.Join([]string{
		localTransferTokenVersion,
		operation,
		assetID,
		strconv.FormatInt(expires.Unix(), 10),
	}, ".")
	signature := hmac.New(sha256.New, s.key)
	_, _ = signature.Write([]byte(payload))
	return payload + "." + base64.RawURLEncoding.EncodeToString(signature.Sum(nil)), expires, nil
}

func (s *transferSigner) Verify(token string, operation string) (transferClaims, error) {
	if s == nil || len(s.key) == 0 || s.now == nil || !validTransferOperation(operation) {
		return transferClaims{}, errInvalidTransferToken
	}
	parts := strings.Split(token, ".")
	if len(parts) != 5 || parts[0] != localTransferTokenVersion || parts[1] != operation {
		return transferClaims{}, errInvalidTransferToken
	}
	if _, err := uuid.Parse(parts[2]); err != nil {
		return transferClaims{}, errInvalidTransferToken
	}
	expiresUnix, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return transferClaims{}, errInvalidTransferToken
	}
	payload := strings.Join(parts[:4], ".")
	want := hmac.New(sha256.New, s.key)
	_, _ = want.Write([]byte(payload))
	received, err := base64.RawURLEncoding.DecodeString(parts[4])
	if err != nil || !hmac.Equal(received, want.Sum(nil)) {
		return transferClaims{}, errInvalidTransferToken
	}
	expires := time.Unix(expiresUnix, 0).UTC()
	if !expires.After(s.now()) {
		return transferClaims{}, errInvalidTransferToken
	}
	return transferClaims{Operation: operation, AssetID: parts[2], Expires: expires}, nil
}

func validTransferOperation(operation string) bool {
	return operation == "upload" || operation == "download"
}
