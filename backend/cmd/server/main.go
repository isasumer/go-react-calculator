// Command server is the calculator API's composition root. For now it is a
// placeholder that answers 404 to every request; routes arrive in B1-02 and
// configuration and lifecycle hardening in B1-03.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Build information, set at link time via -ldflags "-X main.version=…".
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

const (
	defaultAddr       = ":8081"
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 10 * time.Second
)

func main() {
	if err := run(context.Background(), os.Args, os.Getenv, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "server: %v\n", err)
		os.Exit(1)
	}
}

// run starts the HTTP server and blocks until ctx is canceled or SIGINT /
// SIGTERM is received, then shuts down gracefully. It returns nil on a clean
// shutdown. getenv is unused until config parsing lands in B1-03.
func run(ctx context.Context, args []string, _ func(string) string, stdout io.Writer) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.SetOutput(stdout)
	addr := fs.String("addr", defaultAddr, "listen address")
	if err := fs.Parse(args[1:]); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	logger := slog.New(slog.NewJSONHandler(stdout, nil))

	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", *addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", *addr, err)
	}

	srv := &http.Server{
		Handler:           http.NotFoundHandler(),
		ReadHeaderTimeout: readHeaderTimeout,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()

	logger.Info("listening",
		slog.String("addr", ln.Addr().String()),
		slog.String("version", version),
		slog.String("commit", commit),
		slog.String("date", date),
	)

	select {
	case err := <-serveErr:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}
	logger.Info("stopped")
	return nil
}
