package redact

import "testing"

var benchmarkMapResult map[string]any

func BenchmarkMap(b *testing.B) {
	tests := map[string]map[string]any{
		"request_log": {
			"request_id": "req-benchmark",
			"method":     "POST",
			"path":       "/v1/api-keys",
			"status":     201,
			"latency":    "1.2ms",
			"client_ip":  "192.0.2.1",
		},
		"nested_sensitive": {
			"request_id": "req-benchmark",
			"password":   "never-log-this",
			"nested": map[string]any{
				"access_token": "never-log-this-either",
				"outcome":      "success",
			},
		},
	}

	for name, input := range tests {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				benchmarkMapResult = Map(input)
			}
		})
	}
}
