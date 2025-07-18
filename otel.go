package tracing

import (
	"context"
	"errors"
	"fmt"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	ProtocolGRPC = "grpc"
	ProtocolHTTP = "http"
)

type Shutdown func(ctx context.Context) error

type config struct {
	ServiceName string
	Endpoint    string
	Protocol    string
}

func newConfig(opts ...Option) config {
	cfg := config{
		Protocol: ProtocolHTTP,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	return cfg
}

type Option func(*config)

func WithEndpoint(endpoint string) Option {
	return func(c *config) {
		c.Endpoint = endpoint
	}
}

func WithProtocol(protocol string) Option {
	return func(c *config) {
		c.Protocol = protocol
	}
}

func WithServiceName(serviceName string) Option {
	return func(c *config) {
		c.ServiceName = serviceName
	}
}

func validateConfig(c config) error {
	if c.ServiceName == "" {
		return errors.New("tracing lib: ServiceName is required")
	}

	if c.Endpoint == "" {
		return errors.New("tracing lib: Endpoint is required")
	}

	return nil
}

func NewTelemetry(ctx context.Context, opts ...Option) (shutdown Shutdown, err error) {
	cfg := newConfig(opts...)

	err = validateConfig(cfg)
	if err != nil {
		return nil, err
	}

	shutdowns := make([]Shutdown, 0)

	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdowns {
			err = errors.Join(err, fn(ctx))
		}

		shutdowns = nil

		return err
	}

	errHandler := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	res := newResource(cfg.ServiceName)

	tp, err := newTracerProvider(ctx, cfg, res)
	if err != nil {
		errHandler(err)

		return shutdown, err
	}

	shutdowns = append(shutdowns, tp.Shutdown)

	return shutdown, nil
}

// newTracerProvider creates a new tracer provider with the OTLP gRPC exporter.
func newTracerProvider(ctx context.Context, cfg config, res *resource.Resource) (*trace.TracerProvider, error) {
	var (
		exporter *otlptrace.Exporter
		err      error
	)

	switch cfg.Protocol {
	case ProtocolGRPC:
		exporter, err = newGRPCExporter(ctx, cfg)
	case ProtocolHTTP:
		exporter, err = newHTTPExporter(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported OTLP protocol: %s", cfg.Protocol)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	return tp, nil
}

func newGRPCExporter(ctx context.Context, cfg config) (*otlptrace.Exporter, error) {
	conn, err := grpc.NewClient(
		cfg.Endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	return exporter, nil
}

func newHTTPExporter(ctx context.Context, cfg config) (*otlptrace.Exporter, error) {
	exporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpoint(cfg.Endpoint), otlptracehttp.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	return exporter, nil
}

func newResource(serviceName string) *resource.Resource {
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
	)
}
