package commands

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestHealthCheckCommandAcceptsHealthyEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/health/live" {
			response.WriteHeader(http.StatusBadRequest)
			return
		}
		response.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	command := NewHealthCheckCommand()
	if err := command.Run([]string{"--url=" + server.URL + "/health/live", "--timeout=1s"}); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
}

func TestHealthCheckCommandRejectsUnhealthyStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	err := NewHealthCheckCommand().Run([]string{"--url", server.URL, "--timeout", "1s"})
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("Run() error = %v, want status detail", err)
	}
}

func TestHealthCheckCommandHonorsTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		select {
		case <-request.Context().Done():
		case <-time.After(time.Second):
		}
	}))
	t.Cleanup(server.Close)

	err := NewHealthCheckCommand().Run([]string{"--url=" + server.URL, "--timeout=10ms"})
	if err == nil || !strings.Contains(err.Error(), context.DeadlineExceeded.Error()) {
		t.Fatalf("Run() error = %v, want deadline exceeded", err)
	}
}

func TestHealthCheckCommandDefaultsToLoopbackAndConfiguredPort(t *testing.T) {
	t.Setenv("SERVER_PORT", "19025")
	t.Setenv("HEALTHCHECK_URL", "")
	var requestedURL string
	command := &HealthCheckCommand{
		client: &http.Client{
			Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				requestedURL = request.URL.String()
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("ok")),
					Header:     make(http.Header),
					Request:    request,
				}, nil
			}),
		},
		getenv: os.Getenv,
	}

	if err := command.Run(nil); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if requestedURL != "http://127.0.0.1:19025/health/live" {
		t.Fatalf("request URL = %q, want loopback liveness endpoint", requestedURL)
	}
}

func TestHealthCheckCommandExplicitURLDoesNotDependOnServerPort(t *testing.T) {
	t.Setenv("SERVER_PORT", "invalid")
	t.Setenv("HEALTHCHECK_URL", "")
	command := &HealthCheckCommand{
		client: &http.Client{
			Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("ok")),
					Header:     make(http.Header),
					Request:    request,
				}, nil
			}),
		},
		getenv: os.Getenv,
	}

	if err := command.Run([]string{"--url=http://127.0.0.1:19025/health/live"}); err != nil {
		t.Fatalf("Run() error = %v, want explicit URL to bypass SERVER_PORT", err)
	}
}

func TestHealthCheckCommandRejectsInvalidOptions(t *testing.T) {
	tests := [][]string{
		{"--timeout=invalid"},
		{"--timeout=0s"},
		{"--url=ftp://example.com/health"},
		{"--unknown=value"},
	}

	for _, args := range tests {
		if err := NewHealthCheckCommand().Run(args); err == nil {
			t.Fatalf("Run(%q) error = nil, want validation error", args)
		}
	}
}
