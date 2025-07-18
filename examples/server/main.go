package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/mkorobovv/tracing"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdown, err := tracing.NewTelemetry(
		ctx,
		tracing.WithServiceName("example"),     // your service name
		tracing.WithEndpoint("localhost:4317"), // your OTLP exporter host
		tracing.WithProtocol(tracing.ProtocolGRPC), // OTLP protocol (gRPC/HTTP)
	)
	if err != nil {
		panic(err)
	}
	defer shutdown(ctx)

	r := mux.NewRouter()
	r.HandleFunc("/foo/bar", fooBar()).
		Methods("GET")

	server := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("Starting server on :8080")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Server error", slog.Any("err", err))
			os.Exit(1)
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigs:
		logger.Warn("Shutdown signal received")
		if err := server.Shutdown(context.Background()); err != nil {
			logger.Error("Error during shutdown", "err", err)
			os.Exit(1)
		}
		logger.Info("Server gracefully stopped")
		return
	}
}

func fooBar() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		foo(r.Context())

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("foo bar"))
	}
}

func foo(ctx context.Context) {
	ctx, span := tracing.Start(ctx)
	defer span.End()

	time.Sleep(100 * time.Millisecond)

	bar(ctx)
}

func bar(ctx context.Context) {
	ctx, span := tracing.Start(ctx)
	defer span.End()

	time.Sleep(150 * time.Millisecond)
}
