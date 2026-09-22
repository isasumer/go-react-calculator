// Package observability holds the cross-cutting operational concerns: the
// build identity of the running binary and the Prometheus metrics registry.
// The slog logger itself is built in the composition root (cmd/server) from
// the resolved config and injected, and so is [Metrics] — nothing here reads
// or writes a package-level registry.
package observability
