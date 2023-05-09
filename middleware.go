package go_opentelemetry_jaeger

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

func GinMiddleware(endUserIdReceiver EndUserIdReceiver, jaegerHost, jaegerPort string, sampleRate float64) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		tp, err := NewTracerProvider(
			jaegerHost,
			jaegerPort,
			sampleRate,
			semconv.NetPeerPortKey.String(PeerPort),
			semconv.HTTPSchemeKey.String(Scheme),
			semconv.EnduserIDKey.String(endUserIdReceiver.GetEndUserId(ginCtx)),
		)
		if err != nil {
			log.Error().Err(err).Send()
			ginCtx.Next()
			return
		}

		ctx := Extract(ginCtx, propagation.HeaderCarrier(ginCtx.Request.Header))
		ctx, span := NewSpan(ctx, fmt.Sprintf("%s %s", ginCtx.Request.Method, ginCtx.FullPath()))

		InjectSpanInGinContext(ctx, ginCtx)

		ginCtx.Next()

		span.SetAttributes(
			semconv.HTTPURLKey.String(ginCtx.Request.URL.EscapedPath()),
			semconv.HTTPMethodKey.String(ginCtx.Request.Method),
			semconv.HTTPStatusCodeKey.Int(ginCtx.Writer.Status()),
		)
		span.End()

		if err = tp.Shutdown(ctx); err != nil {
			log.Error().Err(err).Send()
		}
	}
}
