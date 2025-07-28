package tracing

import (
	"context"
	"go.opentelemetry.io/otel/propagation"
	"net/http"
	"runtime"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func Start(ctx context.Context) (context.Context, trace.Span) {
	path := functionPath()

	pathParts := strings.Split(path, "/")
	name := pathParts[len(pathParts)-1]

	return otel.Tracer("default_tracer").Start(ctx, name)
}

func functionPath() string {
	pc, _, _, ok := runtime.Caller(2)
	if !ok {
		return "unknown function"
	}

	return runtime.FuncForPC(pc).Name()
}

func AddTracingHook(ctx context.Context, req *http.Request) {
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
}

func StartFromRequest(req *http.Request) (context.Context, trace.Span) {
	ctx := otel.GetTextMapPropagator().Extract(req.Context(), propagation.HeaderCarrier(req.Header))

	path := functionPath()

	pathParts := strings.Split(path, "/")
	name := pathParts[len(pathParts)-1]

	return otel.Tracer("default_tracer").Start(ctx, name)
}
