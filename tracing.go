package go_opentelemetry_jaeger

import (
	"context"
	"github.com/gin-gonic/gin"
	jaeger2 "go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.18.0"
	"go.opentelemetry.io/otel/trace"
	"os"
	"runtime"
)

var (
	AppName         = "my-app"
	Environment     = "development"
	PeerName        = "https://localhost:8080"
	PeerPort        = ":8080"
	MessagingSystem = "rabbitmq"
)

type EndUserIdReceiver interface {
	GetEndUserId(ctx context.Context) string
}

type NoOpEndUserIdReceiver struct{}

func (r *NoOpEndUserIdReceiver) GetEndUserId(ctx context.Context) string {
	return ""
}

func GetMessagingTracerProvider(jaegerHost, jaegerPort string, sampleRate float64, attributes ...attribute.KeyValue) (*tracesdk.TracerProvider, error) {
	attrs := []attribute.KeyValue{
		semconv.MessagingSystemKey.String(MessagingSystem),
		semconv.MessagingDestinationKindTopic,
	}

	return NewTracerProvider(
		jaegerHost,
		jaegerPort,
		sampleRate,
		append(attrs, attributes...)...,
	)
}

func NewTracerProvider(jaegerHost, jaegerPort string, sampleRate float64, attributes ...attribute.KeyValue) (*tracesdk.TracerProvider, error) {
	otel.SetTextMapPropagator(jaeger2.Jaeger{})

	jaegerAgent := jaeger.WithAgentEndpoint(
		jaeger.WithAgentHost(jaegerHost),
		jaeger.WithAgentPort(jaegerPort),
	)

	exporter, err := jaeger.New(jaegerAgent)
	if err != nil {
		return nil, err
	}

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithSampler(tracesdk.TraceIDRatioBased(sampleRate)),
		tracesdk.WithBatcher(exporter),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			getAttributes(attributes)...,
		)),
	)

	otel.SetTracerProvider(tp)

	return tp, nil
}

func getAttributes(attributes []attribute.KeyValue) []attribute.KeyValue {
	hostname, _ := os.Hostname()
	attributes = append(
		attributes,
		semconv.DeploymentEnvironmentKey.String(Environment),
		semconv.NetTransportTCP,
		semconv.ServiceNameKey.String(AppName),
		semconv.NetPeerNameKey.String(PeerName),
		semconv.NetHostNameKey.String(hostname),
		semconv.TelemetrySDKNameKey.String("opentelemetry"),
		semconv.TelemetrySDKLanguageGo,
		semconv.TelemetrySDKVersionKey.String("1.9.0"),
		semconv.ProcessRuntimeNameKey.String("go"),
		semconv.ProcessRuntimeVersionKey.String(runtime.Version()),
	)

	return attributes
}

func NewSpan(ctx context.Context, spanName string, attributes ...attribute.KeyValue) (context.Context, trace.Span) {
	tr := otel.Tracer(AppName)

	return tr.Start(ctx, spanName, trace.WithAttributes(attributes...))
}

func NewSpanFromGinContext(ctx *gin.Context, spanName string, attributes ...attribute.KeyValue) (context.Context, trace.Span) {
	return NewSpan(GetSpanContext(ctx), spanName, attributes...)
}

func Inject(ctx context.Context, carrier propagation.HeaderCarrier) {
	p := otel.GetTextMapPropagator()
	p.Inject(ctx, carrier)
}

func Extract(ctx context.Context, carrier propagation.HeaderCarrier) context.Context {
	p := otel.GetTextMapPropagator()

	return p.Extract(ctx, carrier)
}

func InjectSpanInGinContext(ctx context.Context, gCtx *gin.Context) {
	gCtx.Set(ContextTracerKey, ctx)
}

func GetSpanContext(ctx context.Context) context.Context {
	val := ctx.Value(ContextTracerKey)
	if sp, ok := val.(context.Context); ok {
		return sp
	}

	return ctx
}
