package webhook

import (
	"slices"
	"time"

	"github.com/zgiai/luas/api/internal/domain"
)

type endpointRequest struct {
	Name       string   `json:"name"`
	URL        string   `json:"url"`
	EventTypes []string `json:"event_types"`
}

func (r endpointRequest) input() endpointInput {
	return endpointInput{Name: r.Name, URL: r.URL, EventTypes: slices.Clone(r.EventTypes)}
}

type endpointStatusRequest struct {
	Enabled *bool `json:"enabled"`
}

// EndpointResponse is the secret-free organization management representation.
type EndpointResponse struct {
	ID                   uint                         `json:"id"`
	OrganizationID       uint                         `json:"organization_id"`
	Name                 string                       `json:"name"`
	URL                  string                       `json:"url"`
	EventTypes           []string                     `json:"event_types"`
	Status               domain.WebhookEndpointStatus `json:"status"`
	DisabledReason       string                       `json:"disabled_reason"`
	ConsecutiveFailures  uint8                        `json:"consecutive_failures"`
	Version              uint64                       `json:"version"`
	SecretHint           string                       `json:"secret_hint"`
	SecretVersion        uint64                       `json:"secret_version"`
	PreviousSecretExpiry *time.Time                   `json:"previous_secret_expiry"`
	CreatedAt            time.Time                    `json:"created_at"`
	UpdatedAt            time.Time                    `json:"updated_at"`
}

// EndpointSecretResponse contains one-time plaintext only on create or rotation.
type EndpointSecretResponse struct {
	Endpoint             *EndpointResponse `json:"endpoint"`
	SigningSecret        string            `json:"signing_secret"`
	PreviousSecretExpiry *time.Time        `json:"previous_secret_expiry"`
}

// DeliveryResponse excludes target URLs, payloads, signatures, and response bodies.
type DeliveryResponse struct {
	ID                uint64                       `json:"id"`
	EndpointID        uint                         `json:"endpoint_id"`
	MessageID         string                       `json:"message_id"`
	EventType         string                       `json:"event_type"`
	Status            domain.WebhookDeliveryStatus `json:"status"`
	AttemptCount      uint16                       `json:"attempt_count"`
	ReplayCount       uint16                       `json:"replay_count"`
	HTTPStatus        *int                         `json:"http_status"`
	FailureCode       string                       `json:"failure_code"`
	ResponseTruncated bool                         `json:"response_truncated"`
	AvailableAt       time.Time                    `json:"available_at"`
	DeliveredAt       *time.Time                   `json:"delivered_at"`
	CreatedAt         time.Time                    `json:"created_at"`
	UpdatedAt         time.Time                    `json:"updated_at"`
}

// AttemptResponse is one minimized local execution outcome.
type AttemptResponse struct {
	ID                uint64    `json:"id"`
	DeliveryID        uint64    `json:"delivery_id"`
	Number            uint16    `json:"number"`
	Outcome           string    `json:"outcome"`
	HTTPStatus        *int      `json:"http_status"`
	FailureCode       string    `json:"failure_code"`
	DurationMS        uint64    `json:"duration_ms"`
	ResponseTruncated bool      `json:"response_truncated"`
	StartedAt         time.Time `json:"started_at"`
	CompletedAt       time.Time `json:"completed_at"`
}

func toEndpointResponse(value *domain.WebhookEndpoint) *EndpointResponse {
	if value == nil {
		return nil
	}
	return &EndpointResponse{
		ID:                   value.ID,
		OrganizationID:       value.OrganizationID,
		Name:                 value.Name,
		URL:                  value.URL,
		EventTypes:           slices.Clone(value.EventTypes),
		Status:               value.Status,
		DisabledReason:       value.DisabledReason,
		ConsecutiveFailures:  value.ConsecutiveFailures,
		Version:              value.Version,
		SecretHint:           value.SecretHint,
		SecretVersion:        value.SecretVersion,
		PreviousSecretExpiry: cloneTime(value.PreviousSecretExpiry),
		CreatedAt:            value.CreatedAt,
		UpdatedAt:            value.UpdatedAt,
	}
}

func toEndpointResponses(values []*domain.WebhookEndpoint) []*EndpointResponse {
	result := make([]*EndpointResponse, len(values))
	for index := range values {
		result[index] = toEndpointResponse(values[index])
	}
	return result
}

func toEndpointSecretResponse(value *domain.WebhookEndpointSecret) *EndpointSecretResponse {
	if value == nil {
		return nil
	}
	return &EndpointSecretResponse{
		Endpoint:             toEndpointResponse(value.Endpoint),
		SigningSecret:        value.SigningSecret,
		PreviousSecretExpiry: cloneTime(value.PreviousSecretExpiry),
	}
}

func toDeliveryResponse(value *domain.WebhookDelivery) *DeliveryResponse {
	if value == nil {
		return nil
	}
	return &DeliveryResponse{
		ID:                value.ID,
		EndpointID:        value.EndpointID,
		MessageID:         value.MessageID,
		EventType:         value.EventType,
		Status:            value.Status,
		AttemptCount:      value.AttemptCount,
		ReplayCount:       value.ReplayCount,
		HTTPStatus:        cloneInt(value.HTTPStatus),
		FailureCode:       value.FailureCode,
		ResponseTruncated: value.ResponseTruncated,
		AvailableAt:       value.AvailableAt,
		DeliveredAt:       cloneTime(value.DeliveredAt),
		CreatedAt:         value.CreatedAt,
		UpdatedAt:         value.UpdatedAt,
	}
}

func toDeliveryResponses(values []*domain.WebhookDelivery) []*DeliveryResponse {
	result := make([]*DeliveryResponse, len(values))
	for index := range values {
		result[index] = toDeliveryResponse(values[index])
	}
	return result
}

func toAttemptResponse(value *domain.WebhookAttempt) *AttemptResponse {
	if value == nil {
		return nil
	}
	return &AttemptResponse{
		ID:                value.ID,
		DeliveryID:        value.DeliveryID,
		Number:            value.Number,
		Outcome:           value.Outcome,
		HTTPStatus:        cloneInt(value.HTTPStatus),
		FailureCode:       value.FailureCode,
		DurationMS:        value.DurationMS,
		ResponseTruncated: value.ResponseTruncated,
		StartedAt:         value.StartedAt,
		CompletedAt:       value.CompletedAt,
	}
}

func toAttemptResponses(values []*domain.WebhookAttempt) []*AttemptResponse {
	result := make([]*AttemptResponse, len(values))
	for index := range values {
		result[index] = toAttemptResponse(values[index])
	}
	return result
}
