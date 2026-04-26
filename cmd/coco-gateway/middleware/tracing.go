package middleware

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("coco")

type TraceIDKey string

const TraceIDHeader = "X-Trace-ID"

func generateTraceID() string {
	return fmt.Sprintf("%016x%016x", rand.Uint64(), rand.Uint64())
}

func Tracing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), r.URL.Path,
			trace.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.String()),
				attribute.String("http.remote_addr", r.RemoteAddr),
			),
		)
		defer span.End()

		wrapped := &tracingResponseWriter{ResponseWriter: w, span: span}

		next.ServeHTTP(wrapped, r.WithContext(ctx))

		span.SetAttributes(
			attribute.Int("http.status_code", wrapped.statusCode),
		)

		if wrapped.statusCode >= 400 {
			log.Printf("Request failed: %s %s -> %d", r.Method, r.URL.Path, wrapped.statusCode)
		}
	})
}

func TracingWithTracer(t trace.Tracer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := t.Start(r.Context(), r.URL.Path,
				trace.WithAttributes(
					attribute.String("http.method", r.Method),
					attribute.String("http.url", r.URL.String()),
				),
			)
			defer span.End()

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type tracingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	span       trace.Span
}

func (w *tracingResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasTraceID() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasSpanID() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}
