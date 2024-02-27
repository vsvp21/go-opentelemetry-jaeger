package main

import (
	"context"
	otel "github.com/vsvp21/go-opentelemetry-jaeger"
	"log"
	"net/http"
	"time"
)

func main() {
	otel.PeerName = "127.0.0.1"
	otel.PeerPort = "8080"

	// register tracer provider globally
	tp, err := otel.NewTracerProvider("localhost", "6831", 1)
	if err != nil {
		log.Fatal(err)
	}
	defer tp.Shutdown(context.Background())

	mux := http.NewServeMux()

	h := otel.HTTPMiddleware(otel.HTTPEndUserIdReceiver, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ctx, span := otel.NewSpan(request.Context(), "test")
		defer span.End()

		ctx, span1 := otel.NewSpan(ctx, "test1")

		time.Sleep(time.Second)

		span1.End()
	}))

	mux.Handle("/ping", h)

	http.ListenAndServe(":8082", mux)

}
