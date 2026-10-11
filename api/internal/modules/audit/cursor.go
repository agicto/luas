package audit

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/zgiai/luas/api/internal/domain"
)

const maxCursorLength = 128

// EncodeCursor turns a keyset position into an opaque URL-safe token. Clients pass it back
// unchanged; its layout is not part of the contract.
func EncodeCursor(cursor *domain.AuditLogCursor) string {
	if cursor == nil {
		return ""
	}
	raw := strconv.FormatInt(cursor.CreatedAt.UTC().UnixNano(), 10) + "." + strconv.FormatUint(uint64(cursor.ID), 10)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor parses a token from EncodeCursor. An empty token starts at the newest record.
func DecodeCursor(token string) (*domain.AuditLogCursor, error) {
	if token == "" {
		return nil, nil
	}
	if len(token) > maxCursorLength {
		return nil, fmt.Errorf("%w: cursor is too long", domain.ErrInvalidInput)
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("%w: cursor is malformed", domain.ErrInvalidInput)
	}
	nanos, id, ok := strings.Cut(string(raw), ".")
	if !ok {
		return nil, fmt.Errorf("%w: cursor is malformed", domain.ErrInvalidInput)
	}
	createdAt, timeErr := strconv.ParseInt(nanos, 10, 64)
	recordID, idErr := strconv.ParseUint(id, 10, 64)
	if timeErr != nil || idErr != nil || recordID == 0 || uint64(uint(recordID)) != recordID {
		return nil, fmt.Errorf("%w: cursor is malformed", domain.ErrInvalidInput)
	}
	return &domain.AuditLogCursor{CreatedAt: time.Unix(0, createdAt).UTC(), ID: uint(recordID)}, nil
}

// CursorRequest decodes the keyset position of a history request. A missing or empty cursor starts
// at the newest record. Offset pages were removed, so a page parameter is rejected rather than
// silently ignored.
func CursorRequest(query map[string][]string) (*domain.AuditLogCursor, error) {
	if _, hasPage := query["page"]; hasPage {
		return nil, fmt.Errorf("%w: page is no longer supported; use cursor", domain.ErrInvalidInput)
	}
	values := query["cursor"]
	switch len(values) {
	case 0:
		return nil, nil
	case 1:
		return DecodeCursor(values[0])
	default:
		return nil, fmt.Errorf("%w: cursor must appear once", domain.ErrInvalidInput)
	}
}
