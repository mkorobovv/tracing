# 📡 tracing

`tracing` is a minimal Go package for setting up distributed tracing with OpenTelemetry. It supports both OTLP over HTTP and gRPC and provides a simple API for starting spans using the function name as the span name automatically.

## Features

- Easy setup for OpenTelemetry tracing

- Supports OTLP over HTTP and gRPC

- Automatic span naming based on caller function

- Configurable service name and exporter endpoint

- Graceful shutdown for exporter/trace provider

## Installation
```bash
$ go get github.com/mkorobovv/tracing
```

  ## Quick start

```go
  package main

import (
    "context"
    "log"

    "github.com/mkorobovv/tracing"
)

func main() {
    ctx := context.Background()

    shutdown, err := tracing.NewTelemetry(ctx,
        tracing.WithServiceName("my-service"),
        tracing.WithEndpoint("localhost:4318"), // Default is OTLP/HTTP
        tracing.WithProtocol(tracing.ProtocolHTTP),
    )
    if err != nil {
        log.Fatalf("failed to initialize telemetry: %v", err)
    }
    defer shutdown(ctx)

    ctx, span := tracing.Start(ctx)
    defer span.End()

    doSomething(ctx)
}

func doSomething(ctx context.Context) {
    ctx, span := tracing.Start(ctx)
    defer span.End()

    // Your logic here
}
```
