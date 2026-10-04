package health

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckReportsLatencyInMilliseconds(t *testing.T) {
	h := New()
	h.Register("slow", func(context.Context) CheckResult {
		time.Sleep(15 * time.Millisecond)
		return CheckResult{Status: StatusUp}
	})

	report := h.GetHealth(context.Background())
	result := report.Checks["slow"]
	assert.GreaterOrEqual(t, result.LatencyMS, int64(15))
	assert.Less(t, result.LatencyMS, int64(5000), "milliseconds, not nanoseconds")

	encoded, err := json.Marshal(report)
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"latency_ms":`)
	assert.NotContains(t, string(encoded), `"duration"`)
}
