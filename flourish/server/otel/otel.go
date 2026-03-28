package otel

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/attribute"

	"go.opentelemetry.io/contrib/bridges/otelslog"
)

// Config はOTel初期化設定。
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	OTLPEndpoint   string // e.g. "localhost:4317"
}

// Providers はOTelのプロバイダー群。
type Providers struct {
	TracerProvider *sdktrace.TracerProvider
	LoggerProvider *sdklog.LoggerProvider
}

// Setup はOTelのTracerProviderとLoggerProviderを初期化する。
func Setup(ctx context.Context, cfg Config) (*Providers, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			attribute.String("service.name", cfg.ServiceName),
			attribute.String("service.version", cfg.ServiceVersion),
			attribute.String("deployment.environment", cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	// Trace exporter
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Log exporter
	logExporter, err := otlploggrpc.New(ctx,
		otlploggrpc.WithEndpoint(cfg.OTLPEndpoint),
		otlploggrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create log exporter: %w", err)
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		sdklog.WithResource(res),
	)

	return &Providers{
		TracerProvider: tp,
		LoggerProvider: lp,
	}, nil
}

// Shutdown はプロバイダーをシャットダウンする。
func (p *Providers) Shutdown(ctx context.Context) {
	if p.TracerProvider != nil {
		p.TracerProvider.Shutdown(ctx)
	}
	if p.LoggerProvider != nil {
		p.LoggerProvider.Shutdown(ctx)
	}
}

// NewSlogHandler はOTel LoggerProviderにブリッジするslog.Handlerを返す。
func NewSlogHandler(lp *sdklog.LoggerProvider) slog.Handler {
	return otelslog.NewHandler("crdt-blog", otelslog.WithLoggerProvider(lp))
}

