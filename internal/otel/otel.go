package otel

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/contrib/propagators/autoprop"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var (
	Tracer trace.Tracer
)

func Initialize(ctx context.Context, wg *sync.WaitGroup) {
	if os.Getenv("OTEL_ENABLED") == "" {
		return
	}

	log.Println("initializing OpenTelemetry")

	// Configure Context Propagation to use the default W3C traceparent format
	otel.SetTextMapPropagator(autoprop.NewTextMapPropagator())

	// Configure Trace Export to send spans as OTLP
	texporter, err := autoexport.NewSpanExporter(ctx)
	if err != nil {
		log.Panic(err)
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(texporter))

	otel.SetTracerProvider(tp)
	Tracer = otel.Tracer("echo")

	log.Println("OpenTelemetry initialized")

	wg.Add(1)
	go func() {
		<-ctx.Done()
		log.Println("shutting down OpenTelemetry")
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			log.Fatalf("failed to shutdown tracer provider: %v", err)
		}
		wg.Done()
	}()

}
