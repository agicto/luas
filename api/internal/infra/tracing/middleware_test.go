package tracing

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestMiddlewareExportsRouteShapeWithoutConcretePathOrErrorMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() {
		otel.SetTracerProvider(previous)
		require.NoError(t, provider.Shutdown(context.Background()))
	})

	engine := gin.New()
	engine.Use(Middleware("luas-test"))
	engine.GET("/accounts/:account", func(c *gin.Context) {
		_ = c.Error(assert.AnError)
		c.Status(http.StatusInternalServerError)
	})
	request := httptest.NewRequest(
		http.MethodGet,
		"/accounts/private-customer?access_token=trace-secret",
		nil,
	)
	response := httptest.NewRecorder()

	engine.ServeHTTP(response, request)

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "GET /accounts/:account", span.Name())

	attributes := make(map[string]string, len(span.Attributes()))
	for _, item := range span.Attributes() {
		attributes[string(item.Key)] = item.Value.String()
	}
	assert.Equal(t, "/accounts/:account", attributes["http.route"])
	assert.NotContains(t, attributes, "url.path")

	var exported strings.Builder
	for key, value := range attributes {
		exported.WriteString(key)
		exported.WriteString(value)
	}
	assert.NotContains(t, exported.String(), "private-customer")
	assert.NotContains(t, exported.String(), "trace-secret")
	assert.NotContains(t, exported.String(), assert.AnError.Error())
}

func TestRecordErrorExportsTypeWithoutFreeFormMessage(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() { require.NoError(t, provider.Shutdown(context.Background())) })

	ctx, span := provider.Tracer("luas-test").Start(context.Background(), "database.operation")
	RecordError(ctx, errors.New("database-free-form-secret"))
	span.End()

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	assert.Equal(t, codes.Error, spans[0].Status().Code)
	assert.Empty(t, spans[0].Status().Description)
	assert.Empty(t, spans[0].Events())

	attributes := make(map[string]string, len(spans[0].Attributes()))
	for _, item := range spans[0].Attributes() {
		attributes[string(item.Key)] = item.Value.String()
	}
	assert.Equal(t, "*errors.errorString", attributes["error.type"])
	assert.NotContains(t, attributes, "database-free-form-secret")
}
