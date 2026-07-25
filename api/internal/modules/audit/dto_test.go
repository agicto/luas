package audit

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin/binding"
	"github.com/stretchr/testify/assert"
)

func TestAuditLogListRequestBoundsFilters(t *testing.T) {
	assert.NoError(t, binding.Validator.ValidateStruct(&AuditLogListRequest{
		Action:     "update",
		Resource:   "users.profile",
		Method:     "PATCH",
		RequestID:  "req_123",
		StatusCode: 200,
	}))

	invalid := []*AuditLogListRequest{
		{Action: strings.Repeat("a", 121)},
		{Resource: strings.Repeat("r", 181)},
		{Method: strings.Repeat("M", 11)},
		{RequestID: strings.Repeat("q", 81)},
		{StatusCode: 99},
		{StatusCode: 600},
	}
	for _, request := range invalid {
		assert.Error(t, binding.Validator.ValidateStruct(request))
	}
}
