package main

import (
	"context"
	otel "github.com/vsvp21/go-opentelemetry-jaeger"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	otel.PeerName = "127.0.0.1"
	otel.PeerPort = "8080"

	// register tracer provider globally
	tp, err := otel.NewTracerProvider("localhost", "6381", 1)
	if err != nil {
		log.Fatal(err)
	}
	defer tp.Shutdown(context.Background())

	// serve
	r := gin.Default()
	r.GET("/ping", otel.GinMiddleware(otel.GinEndUserIdReceiver), func(c *gin.Context) {
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
