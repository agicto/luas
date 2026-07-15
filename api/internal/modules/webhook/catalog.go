package webhook

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"

	"github.com/zgiai/luas/api/internal/domain"
)

const (
	maxWebhookDefinitions    = 128
	maxWebhookEventTypeBytes = 100
	maxWebhookPayloadBytes   = 64 * 1024
	maxWebhookSourceBytes    = 64
	maxWebhookEventIDBytes   = 128
)

var (
	webhookEventTypePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)+$`)
	webhookSourcePattern    = regexp.MustCompile(`^[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)*$`)
	webhookEventIDPattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_:-]*$`)
)

// Definition owns one stable event type and exact payload schema.
type Definition struct {
	Type            string
	ValidatePayload func(json.RawMessage) error
}

// Catalog is one finite immutable event type snapshot.
type Catalog struct {
	ordered []Definition
	byType  map[string]Definition
}

// NewDefaultCatalog returns only the starter-owned delivery test event.
func NewDefaultCatalog() (*Catalog, error) {
	return NewCatalog(Definition{
		Type:            "webhook.test",
		ValidatePayload: validateTestPayload,
	})
}

// NewCatalog validates and freezes event definitions.
func NewCatalog(definitions ...Definition) (*Catalog, error) {
	if len(definitions) == 0 {
		return nil, fmt.Errorf("at least one webhook event definition is required")
	}
	if len(definitions) > maxWebhookDefinitions {
		return nil, fmt.Errorf("webhook catalog exceeds %d definitions", maxWebhookDefinitions)
	}
	catalog := &Catalog{
		ordered: make([]Definition, 0, len(definitions)),
		byType:  make(map[string]Definition, len(definitions)),
	}
	for _, definition := range definitions {
		if !validWebhookEventType(definition.Type) || definition.ValidatePayload == nil {
			return nil, fmt.Errorf("webhook event definition %q is invalid", definition.Type)
		}
		if _, exists := catalog.byType[definition.Type]; exists {
			return nil, fmt.Errorf("duplicate webhook event definition %q", definition.Type)
		}
		catalog.byType[definition.Type] = definition
		catalog.ordered = append(catalog.ordered, definition)
	}
	slices.SortFunc(catalog.ordered, func(left, right Definition) int {
		if left.Type < right.Type {
			return -1
		}
		if left.Type > right.Type {
			return 1
		}
		return 0
	})
	return catalog, nil
}

// Types returns the stable catalog keys in lexical order.
func (c *Catalog) Types() []string {
	if c == nil {
		return nil
	}
	types := make([]string, len(c.ordered))
	for index := range c.ordered {
		types[index] = c.ordered[index].Type
	}
	return types
}

// Contains reports whether an exact event type is registered.
func (c *Catalog) Contains(eventType string) bool {
	if c == nil {
		return false
	}
	_, ok := c.byType[eventType]
	return ok
}

// Normalize validates one payload and returns deterministic compact JSON.
func (c *Catalog) Normalize(eventType string, raw json.RawMessage) (string, error) {
	if c == nil {
		return "", domainEventTypeError()
	}
	definition, exists := c.byType[eventType]
	if !exists {
		return "", domainEventTypeError()
	}
	canonical, err := canonicalJSONObject(raw)
	if err != nil || definition.ValidatePayload(json.RawMessage(canonical)) != nil {
		return "", fmt.Errorf("invalid webhook payload: %w", errWebhookPayload)
	}
	return canonical, nil
}

func validWebhookEventType(value string) bool {
	return len(value) > 0 && len(value) <= maxWebhookEventTypeBytes && webhookEventTypePattern.MatchString(value)
}

func validWebhookSource(value string) bool {
	return len(value) > 0 && len(value) <= maxWebhookSourceBytes && webhookSourcePattern.MatchString(value)
}

func validWebhookEventID(value string) bool {
	return len(value) > 0 && len(value) <= maxWebhookEventIDBytes && webhookEventIDPattern.MatchString(value)
}

var errWebhookPayload = errors.New("webhook payload is invalid")

func domainEventTypeError() error {
	return fmt.Errorf("unknown webhook event type: %w", domain.ErrWebhookInvalidEventType)
}

func canonicalJSONObject(raw json.RawMessage) (string, error) {
	if len(raw) < 2 || len(raw) > maxWebhookPayloadBytes {
		return "", errWebhookPayload
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := rejectDuplicateJSONKeys(decoder); err != nil {
		return "", errWebhookPayload
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return "", errWebhookPayload
	}

	decoder = json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value map[string]any
	if err := decoder.Decode(&value); err != nil || value == nil {
		return "", errWebhookPayload
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return "", errWebhookPayload
	}
	encoded, err := json.Marshal(value)
	if err != nil || len(encoded) > maxWebhookPayloadBytes {
		return "", errWebhookPayload
	}
	return string(encoded), nil
}

func rejectDuplicateJSONKeys(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	return consumeJSONToken(decoder, token)
}

func consumeJSONToken(decoder *json.Decoder, token json.Token) error {
	delimiter, isDelimiter := token.(json.Delim)
	if !isDelimiter {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errWebhookPayload
			}
			if _, duplicate := seen[key]; duplicate {
				return errWebhookPayload
			}
			seen[key] = struct{}{}
			valueToken, err := decoder.Token()
			if err != nil {
				return err
			}
			if err := consumeJSONToken(decoder, valueToken); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim('}') {
			return errWebhookPayload
		}
	case '[':
		for decoder.More() {
			valueToken, err := decoder.Token()
			if err != nil {
				return err
			}
			if err := consumeJSONToken(decoder, valueToken); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim(']') {
			return errWebhookPayload
		}
	default:
		return errWebhookPayload
	}
	return nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var trailing any
	err := decoder.Decode(&trailing)
	if errors.Is(err, io.EOF) {
		return nil
	}
	return errWebhookPayload
}

func validateTestPayload(raw json.RawMessage) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var payload struct {
		OrganizationID uint `json:"organization_id"`
		EndpointID     uint `json:"endpoint_id"`
	}
	if err := decoder.Decode(&payload); err != nil || ensureJSONEOF(decoder) != nil {
		return errWebhookPayload
	}
	if payload.OrganizationID == 0 || payload.EndpointID == 0 {
		return errWebhookPayload
	}
	return nil
}
