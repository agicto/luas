package tracing

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// Middleware returns a Gin middleware for HTTP tracing
func Middleware(serviceName string) gin.HandlerFunc {
	tracer := otel.Tracer(serviceName)
	propagator := otel.GetTextMapPropagator()

	return func(c *gin.Context) {
		// Extract trace context from incoming request
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		// Route templates keep span names bounded and avoid exporting path values.
		route := c.FullPath()
		spanName := c.Request.Method
		attributes := []attribute.KeyValue{
			semconv.HTTPRequestMethodKey.String(c.Request.Method),
			semconv.URLScheme(c.Request.URL.Scheme),
			semconv.ServerAddress(c.Request.Host),
			semconv.UserAgentOriginal(c.Request.UserAgent()),
			attribute.String("http.client_ip", c.ClientIP()),
		}
		if route != "" {
			spanName = fmt.Sprintf("%s %s", c.Request.Method, route)
			attributes = append(attributes, semconv.HTTPRoute(route))
		}

		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(attributes...),
		)
		defer span.End()

		// Store span in context
		c.Request = c.Request.WithContext(ctx)

		// Process request
		c.Next()

		// Record response attributes
		status := c.Writer.Status()
		span.SetAttributes(
			semconv.HTTPResponseStatusCode(status),
			attribute.Int("http.response_size", c.Writer.Size()),
		)

		// Record errors
		if len(c.Errors) > 0 {
			errorTypes := make([]string, 0, len(c.Errors))
			for _, err := range c.Errors {
				errorTypes = append(errorTypes, fmt.Sprintf("%T", err.Err))
			}
			span.SetAttributes(
				attribute.Int("http.error_count", len(c.Errors)),
				attribute.StringSlice("error.types", errorTypes),
			)
		}

		// Set span status based on HTTP status code
		if status >= 400 {
			span.SetAttributes(attribute.Bool("error", true))
		}
	}
}

// InjectTraceID injects trace ID into response headers
func InjectTraceID() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		span := trace.SpanFromContext(c.Request.Context())
		if span.SpanContext().IsValid() {
			c.Header("X-Trace-ID", span.SpanContext().TraceID().String())
		}
	}
}
