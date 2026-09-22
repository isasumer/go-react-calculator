package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/isasumer/go-react-calculator/backend/internal/config"
)

// healthcheckTimeout bounds the probe end to end. Docker's own --timeout kills
// a hung check eventually, but a deadline inside the process turns "no answer"
// into a clear exit-1 with a reason instead of a SIGKILL.
const healthcheckTimeout = 2 * time.Second

// healthcheckURL is the readiness URL the in-container probe calls. It always
// targets the loopback interface — the probe runs inside the container's own
// network namespace, so HOST (which may be 0.0.0.0) is irrelevant and only the
// port matters. An unparsable PORT falls back to the default rather than
// failing the probe for a reason the probe cannot fix; config.Load is the
// place that rejects bad values, at startup.
func healthcheckURL(getenv func(string) string) string {
	port := strconv.Itoa(config.DefaultPort)
	if raw := strings.TrimSpace(getenv("PORT")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 65535 {
			port = raw
		}
	}
	return "http://" + net.JoinHostPort("127.0.0.1", port) + "/readyz"
}

// healthcheck GETs url and returns nil when the process behind it reports
// ready. This is what `server -healthcheck` runs: the distroless final image
// has no shell and no curl, so the binary probes itself.
//
// The client is built here rather than reused from http.DefaultClient so the
// probe ignores any proxy environment and never keeps a connection open — it
// is a one-shot process.
func healthcheck(ctx context.Context, url string) error {
	ctx, cancel := context.WithTimeout(ctx, healthcheckTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("healthcheck: %w", err)
	}
	client := &http.Client{Transport: &http.Transport{DisableKeepAlives: true}}
	defer client.CloseIdleConnections()

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("healthcheck: %w", err)
	}
	defer resp.Body.Close()
	// Drain a bounded amount so the connection can be reused/closed cleanly.
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck: GET %s: %s", url, resp.Status)
	}
	return nil
}
