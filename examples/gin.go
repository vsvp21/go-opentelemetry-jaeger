package main

import (
	"context"
	tracer "go-opentelemetry-jaeger"
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
	r.GET("/ping", tracer.GinMiddleware(&endUserReceiverFromCtx{}, "ykim.made.kz", "6831"), func(c *gin.Context) {
		_, span := tracer.NewSpanFromGinContext(c, "test")
		time.Sleep(time.Second)
		span.End()

		ctx, span := tracer.NewSpanFromGinContext(c, "test1")
		_, span1 := tracer.NewSpan(ctx, "test1-inner")
		time.Sleep(time.Second)
		span1.End()
		span.End()
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.Run()
}
