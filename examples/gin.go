package main

import (
	"context"
	otel "github.com/vsvp21/go-opentelemetry-jaeger"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type endUserReceiverFromCtx struct{}

func (r *endUserReceiverFromCtx) GetEndUserId(ctx context.Context) string {
	return ctx.(*gin.Context).GetHeader("X-User-Id")
}

func main() {
	r := gin.Default()
	r.GET("/ping", otel.GinMiddleware(&endUserReceiverFromCtx{}, "127.0.0.1", "6831", 0.5), func(c *gin.Context) {
		_, span := otel.NewSpanFromGinContext(c, "test")
		time.Sleep(time.Second)
		span.End()

		ctx, span := otel.NewSpanFromGinContext(c, "test1")
		_, span1 := otel.NewSpan(ctx, "test1-inner")
		time.Sleep(time.Second)
		span1.End()
		span.End()
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.Run()
}
