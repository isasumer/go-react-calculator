// Package observability holds the cross-cutting operational concerns: the
// build identity of the running binary (B1-03) and, from B1-05, the
// Prometheus metrics registry. The slog logger itself is built in the
// composition root (cmd/server) from the resolved config and injected.
package observability
