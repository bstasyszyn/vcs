/*
   Copyright SecureKey Technologies Inc.

   This file contains software code that is the intellectual property of SecureKey.
   SecureKey reserves all rights in the code and you may not use it without
	 written permission from SecureKey.
*/

package tracing

import (
	"context"
	"fmt"
	"os"

	"github.com/trustbloc/logutil-go/pkg/log"
	jaegerprop "go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

var logger = log.New("tracing")

// ProviderType specifies the type of the tracer provider.
type ProviderType = string

const (
	// ProviderNone indicates that tracing is disabled.
	ProviderNone ProviderType = ""
	// ProviderJaeger indicates that tracing data should be in Jaeger format.
	ProviderJaeger ProviderType = "JAEGER"

	tracerName = "https://github.com/trustbloc/vcs"
)

// Initialize creates and registers globally a new tracer Provider.
func Initialize(provider, serviceName, url string) (trace.TracerProvider, trace.Tracer, error) {
	if provider == ProviderNone {
		return nil, trace.NewNoopTracerProvider().Tracer(""), nil
	}

	var tp *tracesdk.TracerProvider

	switch provider {
	case ProviderJaeger:
		otel.SetTextMapPropagator(jaegerprop.Jaeger{})

		var err error

		tp, err = newJaegerTracerProvider(serviceName, url)
		if err != nil {
			return nil, nil, fmt.Errorf("create new tracer provider: %w", err)
		}
	default:
		return nil, nil, fmt.Errorf("unsupported tracing provider: %s", provider)
	}

	// Register the TracerProvider as the global so any imported
	// instrumentation in the future will default to using it.
	otel.SetTracerProvider(tp)

	return &otelTracerProvider{TracerProvider: tp}, tp.Tracer(tracerName), nil
}

// newJaegerTracerProvider returns an OpenTelemetry Provider configured to use
// the Jaeger exporter that will send spans to the provided url. The returned
// Provider will also use a Resource configured with all the information
// about the application.
func newJaegerTracerProvider(serviceName, url string) (*tracesdk.TracerProvider, error) {
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(url)))
	if err != nil {
		return nil, fmt.Errorf("create jaeger collector: %w", err)
	}

	return tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
			semconv.ProcessPIDKey.Int(os.Getpid()),
		)),
	), nil
}

type noopTracerProvider struct {
	trace.TracerProvider
}

func (tp *noopTracerProvider) Start() {}

func (tp *noopTracerProvider) Stop() {}

type otelTracerProvider struct {
	*tracesdk.TracerProvider
}

func (tp *otelTracerProvider) Start() {}

func (tp *otelTracerProvider) Stop() {
	if err := tp.TracerProvider.Shutdown(context.Background()); err != nil {
		logger.Warn("Error shutting down tracer provider", log.WithError(err))
	}
}
