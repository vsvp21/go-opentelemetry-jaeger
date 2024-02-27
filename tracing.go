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
	"net/http"
	"os"
	"runtime"
)

const ContextTracerKey = "Tracer-context"

type IdentityKey string

var (
	// AppName service name for jaeger ui
	AppName = "my-app"
	// Environment env for jaeger ui
	Environment = "development"
	// PeerName peer name according to specs
	PeerName = "https://localhost"
	// PeerPort peer port according to specs
	PeerPort = ":8080"
	// MessagingSystem message broker type according to specification
	MessagingSystem = "rabbitmq"
	// UserIdentityKey key that represents user identity in http headers
	UserIdentityKey IdentityKey = "X-User-Id"
)

// EndUserIdReceiver retrieves user id from context
type EndUserIdReceiver func(ctx context.Context) string

// NoOpEndUserId ...
func NoOpEndUserId(ctx context.Context) string {
	return ""
}

func GinEndUserIdReceiver(ctx context.Context) string {
	return ctx.(*gin.Context).GetHeader(string(UserIdentityKey))
}

func HTTPEndUserIdReceiver(ctx context.Context) string {
	return ctx.Value(UserIdentityKey).(string)
}

// NewTracerProvider creates new tracer provider with jaeger exporter
// registers tracer provider globally
// registers jaeger text map propagator globally
// also sets basic otel attributes accoding to specification
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

	hostname, _ := os.Hostname()
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithSampler(tracesdk.TraceIDRatioBased(sampleRate)),
		tracesdk.WithBatcher(exporter),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			append(
				attributes,
				semconv.DeploymentEnvironmentKey.String(Environment),
				semconv.NetTransportTCP,
				semconv.ServiceNameKey.String(AppName),
				semconv.NetPeerNameKey.String(PeerName),
				semconv.NetHostNameKey.String(hostname),
				semconv.TelemetrySDKNameKey.String("opentelemetry"),
				semconv.TelemetrySDKLanguageGo,
				semconv.TelemetrySDKVersionKey.String("1.19.0"),
				semconv.ProcessRuntimeNameKey.String("go"),
				semconv.ProcessRuntimeVersionKey.String(runtime.Version()),
			)...,
		)),
	)

	otel.SetTracerProvider(tp)

	return tp, nil
}

// NewSpan creates new span with additional attributes if passed
func NewSpan(ctx context.Context, spanName string, attributes ...attribute.KeyValue) (context.Context, trace.Span) {
	tr := otel.Tracer(AppName)

	return tr.Start(ctx, spanName, trace.WithAttributes(attributes...))
}

// NewRabbitMQSpan creates new span for rabbitmq listener with additional attributes if passed
func NewRabbitMQSpan(ctx context.Context, spanName, consumerName, routingKey string, attributes ...attribute.KeyValue) (context.Context, trace.Span) {
	attrs := []attribute.KeyValue{
		semconv.MessagingSystemKey.String(MessagingSystem),
		semconv.MessagingDestinationKindTopic,
		semconv.MessagingOperationKey.String(consumerName),
		semconv.MessagingRabbitmqDestinationRoutingKey(routingKey),
	}

	return NewSpan(
		ctx,
		spanName,
		append(attrs, attributes...)...,
	)

}

// NewSpanFromGinContext creates new span from gin context with additional attributes if passed
func NewSpanFromGinContext(ctx *gin.Context, spanName string, attributes ...attribute.KeyValue) (context.Context, trace.Span) {
	return NewSpan(GetSpanContext(ctx), spanName, attributes...)
}

// Inject injects trace id data to HeaderCarrier, needed to pass trace id further to next service
func Inject(ctx context.Context, carrier propagation.HeaderCarrier) {
	p := otel.GetTextMapPropagator()
	p.Inject(ctx, carrier)
}

// Extract extracts trace id data from HeaderCarrier, needed to get trace from request
func Extract(ctx context.Context, carrier propagation.HeaderCarrier) context.Context {
	p := otel.GetTextMapPropagator()

	return p.Extract(ctx, carrier)
}

// InjectSpanInGinContext injects trace id to gin context
func InjectSpanInGinContext(ctx context.Context, gCtx *gin.Context) {
	gCtx.Set(ContextTracerKey, ctx)
}

// GetSpanContext get trace id from context
func GetSpanContext(ctx context.Context) context.Context {
	val := ctx.Value(ContextTracerKey)
	if sp, ok := val.(context.Context); ok {
		return sp
	}

	return ctx
}

// ResponseWriter helps to intercept http status code from http.ResponseWriter
type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// NewResponseWriter creates a new responseWriter instance.
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	// Default the status code to 200, since that's what net/http defaults to.
	return &ResponseWriter{w, http.StatusOK}
}

// WriteHeader captures the status code for logging and calls the underlying WriteHeader method.
func (rw *ResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
