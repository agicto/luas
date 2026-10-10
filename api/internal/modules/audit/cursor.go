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

// CursorRequest reports whether the request asked for keyset pagination and decodes its position.
// A present but empty cursor parameter requests the first keyset page.
func CursorRequest(query map[string][]string) (cursor *domain.AuditLogCursor, keyset bool, err error) {
	values, keyset := query["cursor"]
	if !keyset {
		return nil, false, nil
	}
	if _, hasPage := query["page"]; hasPage {
		return nil, true, fmt.Errorf("%w: cursor cannot be combined with page", domain.ErrInvalidInput)
	}
	if len(values) != 1 {
		return nil, true, fmt.Errorf("%w: cursor must appear once", domain.ErrInvalidInput)
	}
	cursor, err = DecodeCursor(values[0])
	return cursor, true, err
}
