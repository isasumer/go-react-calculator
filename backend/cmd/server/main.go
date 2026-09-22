// Command server is the calculator API's composition root: it resolves the
// configuration from the environment, builds the logger, the operations
// registry, the HTTP handler and the middleware chain around it, serves them,
// and drains cleanly on a signal.
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
	"sync/atomic"
	"syscall"
	"time"

	"github.com/isasumer/go-react-calculator/backend/internal/calc"
	"github.com/isasumer/go-react-calculator/backend/internal/config"
	"github.com/isasumer/go-react-calculator/backend/internal/httpapi"
	"github.com/isasumer/go-react-calculator/backend/internal/middleware"
	"github.com/isasumer/go-react-calculator/backend/internal/observability"
)

// main is a thin wrapper: it owns the process (signals, exit code) and
// nothing else. Signal handling lives here rather than in run so a test can
// drive run with a context of its own.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Args, os.Getenv, os.Stdout)
	stop()
	if err != nil {
		fmt.Fprintf(os.Stderr, "server: %v\n", err)
		os.Exit(1)
	}
}

// run is the whole program. Everything it touches is an argument, so tests
// drive it exactly as main does. It blocks until ctx is canceled, then takes
// the process out of rotation and drains, and returns the first error it hit
// (nil after a clean shutdown).
func run(ctx context.Context, args []string, getenv func(string) string, stdout io.Writer) error {
	build := observability.Build()

	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.SetOutput(stdout)
	printVersion := fs.Bool("version", false, "print build information and exit")
	probe := fs.Bool("healthcheck", false, "probe the local /readyz endpoint and exit 0 when ready")
	if err := fs.Parse(args[1:]); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	if *printVersion {
		_, err := fmt.Fprintf(stdout, "server %s\n", build)
		return err
	}
	// The probe is its own tiny program sharing the binary: the distroless
	// image has no shell and no curl, so Docker's HEALTHCHECK runs this. It
	// deliberately runs before config.Load — a probe must not fail for a
	// reason unrelated to the running server's readiness.
	if *probe {
		return healthcheck(ctx, healthcheckURL(getenv))
	}

	cfg, err := config.Load(getenv)
	if err != nil {
		return err
	}
	logger := newLogger(stdout, cfg)

	// One registry for the process, owned here and passed down: no package
	// touches a global registerer, and a test builds its own.
	metrics := observability.NewMetrics(observability.NewRegistry(), build)

	// The readiness flag is owned by the lifecycle below: the probe reports
	// not-ready from the moment shutdown starts, before the listener closes.
	var ready atomic.Bool
	handler := httpapi.NewHandler(calc.NewRegistry(), logger,
		httpapi.WithReadiness(ready.Load),
		httpapi.WithBuildInfo(build),
		httpapi.WithMetrics(metrics),
	)

	srv := &http.Server{
		Handler:           chain(handler.Routes(), cfg, logger, metrics),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	// Listen before serving: a bind failure becomes an error instead of a
	// log line, and PORT=0 has resolved to a real port by the time the
	// startup line reports the address.
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", cfg.Addr())
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.Addr(), err)
	}

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()
	ready.Store(true)

	logger.Info("listening",
		slog.String("addr", ln.Addr().String()),
		slog.String("version", build.Version),
		slog.String("commit", build.Commit),
		slog.String("buildDate", build.BuildDate),
		slog.String("config", cfg.String()),
	)

	select {
	case err := <-serveErr:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}

	ready.Store(false)
	return drain(ctx, srv, cfg, logger, serveErr)
}

// chain wraps the router in the middleware chain. The order is the order a
// request travels, and it is the whole reason middleware.Chain takes a list
// instead of being a pile of nested calls: this is the one place the request
// lifecycle is written down, so it can be read.
func chain(router http.Handler, cfg config.Config, logger *slog.Logger, metrics *observability.Metrics) http.Handler {
	return middleware.Chain(router,
		middleware.Recover(logger),
		middleware.RequestID(),
		middleware.Logger(logger),
		middleware.Timeout(cfg.RequestTimeout),
		middleware.SecurityHeaders(),
		middleware.CORS(cfg.CORSAllowedOrigins),
		middleware.RateLimit(middleware.RateLimitConfig{
			RPS:               cfg.RateLimitRPS,
			Burst:             cfg.RateLimitBurst,
			TrustProxyHeaders: cfg.TrustProxyHeaders,
		}),
		// Innermost, between the rate limiter and the router: a rejected
		// request costs no observation, and the route label is the pattern
		// the router matched rather than the raw path. Chain skips a nil
		// entry, so a caller without metrics still gets a working chain.
		middleware.Metrics(metrics),
	)
}

// drain stops the server the way a rolling deploy wants it: readiness is
// already false, so the pre-stop delay gives a load balancer time to notice
// before the listener closes, and Shutdown then lets in-flight requests
// finish within SHUTDOWN_TIMEOUT.
func drain(ctx context.Context, srv *http.Server, cfg config.Config, logger *slog.Logger, serveErr <-chan error) error {
	start := time.Now()
	logger.Info("shutting down",
		slog.Duration("preStopDelay", cfg.PreStopDelay),
		slog.Duration("timeout", cfg.ShutdownTimeout))
	time.Sleep(cfg.PreStopDelay)

	// ctx is already canceled; the drain gets a deadline of its own.
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.ShutdownTimeout)
	defer cancel()

	err := srv.Shutdown(shutdownCtx)
	if err != nil {
		err = fmt.Errorf("shutdown: %w", err)
	}
	// Serve always returns once the listener is closed.
	if serr := <-serveErr; !errors.Is(serr, http.ErrServerClosed) && err == nil {
		err = fmt.Errorf("serve: %w", serr)
	}
	logger.Info("stopped", slog.Duration("drain", time.Since(start)))
	return err
}

// newLogger builds the logger the process writes through: level and format
// come from the environment, the destination is the injected writer (stdout
// for the binary, a pipe in tests).
func newLogger(w io.Writer, cfg config.Config) *slog.Logger {
	opts := &slog.HandlerOptions{Level: cfg.LogLevel}
	var h slog.Handler
	switch cfg.LogFormat {
	case config.FormatText:
		h = slog.NewTextHandler(w, opts)
	default:
		h = slog.NewJSONHandler(w, opts)
	}
	return slog.New(h)
}
