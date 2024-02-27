package go_opentelemetry_jaeger

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"net/http"
)

const Scheme = "https"

// GinMiddleware middleware for gin gonic
// Starts and Ends span for each request
func GinMiddleware(endUserIdReceiver EndUserIdReceiver) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		ctx := Extract(ginCtx, propagation.HeaderCarrier(ginCtx.Request.Header))

		ctx, span := NewSpan(ctx, fmt.Sprintf("%s %s", ginCtx.Request.Method, ginCtx.FullPath()))

		InjectSpanInGinContext(ctx, ginCtx)

		ginCtx.Next()

		attrs := []attribute.KeyValue{
			semconv.HTTPURLKey.String(ginCtx.Request.URL.EscapedPath()),
			semconv.HTTPMethodKey.String(ginCtx.Request.Method),
			semconv.HTTPStatusCodeKey.Int(ginCtx.Writer.Status()),
			semconv.NetPeerPortKey.String(PeerPort),
			semconv.HTTPSchemeKey.String(Scheme),
		}

		if endUserIdReceiver != nil {
			attrs = append(attrs, semconv.EnduserIDKey.String(endUserIdReceiver(ginCtx)))
		}

		span.SetAttributes(attrs...)
		span.End()
	}
}

func HTTPMiddleware(endUserIdReceiver EndUserIdReceiver, next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		ctx := Extract(req.Context(), propagation.HeaderCarrier(req.Header))

		ctx, span := NewSpan(ctx, fmt.Sprintf("%s %s", req.Method, req.URL.Path))

		req = req.WithContext(context.WithValue(ctx, UserIdentityKey, req.Header.Get(string(UserIdentityKey))))

		wrapper := NewResponseWriter(rw)
		next.ServeHTTP(wrapper, req)

		attrs := []attribute.KeyValue{
			semconv.HTTPURLKey.String(req.URL.EscapedPath()),
			semconv.HTTPMethodKey.String(req.Method),
			semconv.HTTPStatusCodeKey.Int(wrapper.statusCode),
			semconv.NetPeerPortKey.String(PeerPort),
			semconv.HTTPSchemeKey.String(Scheme),
		}

		if endUserIdReceiver != nil {
			attrs = append(attrs, semconv.EnduserIDKey.String(endUserIdReceiver(req.Context())))
		}

		span.SetAttributes(attrs...)
		span.End()
	})
}
