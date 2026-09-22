package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"time"
)

// Default values for every variable [Load] reads. They are the values a
// developer gets with an empty environment: a local-friendly, safe server.
const (
	DefaultHost              = "0.0.0.0"
	DefaultPort              = 8081
	DefaultLogLevel          = slog.LevelInfo
	DefaultLogFormat         = FormatJSON
	DefaultReadHeaderTimeout = 5 * time.Second
	DefaultReadTimeout       = 10 * time.Second
	DefaultWriteTimeout      = 10 * time.Second
	DefaultIdleTimeout       = 60 * time.Second
	DefaultShutdownTimeout   = 15 * time.Second
	DefaultRequestTimeout    = 5 * time.Second
	DefaultPreStopDelay      = 0 * time.Second
	DefaultRateLimitRPS      = 20
	DefaultRateLimitBurst    = 40
	DefaultMaxBodyBytes      = 4096
	DefaultTrustProxyHeaders = false
)

// maxPort is the highest TCP port. Port 0 is allowed: it asks the kernel for
// a free port, which is how the tests run the server.
const maxPort = 65535

// LogFormat selects the slog handler the server logs through.
type LogFormat string

// Supported log formats.
const (
	FormatJSON LogFormat = "json"
	FormatText LogFormat = "text"
)

// Config is the resolved configuration of one server process. Fields are
// grouped as they are used: address, logging, CORS, server timeouts,
// lifecycle, and the limits the middleware chain (B1-04) applies.
type Config struct {
	Host string
	Port int

	LogLevel  slog.Level
	LogFormat LogFormat

	// CORSAllowedOrigins is empty by default: in production the frontend is
	// served from the same origin (ADR-0009), so no origin is allowed.
	CORSAllowedOrigins []string

	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration

	// ShutdownTimeout bounds the drain; PreStopDelay is the pause between
	// flipping readiness off and closing the listener, so a load balancer
	// stops routing before the process stops accepting.
	ShutdownTimeout time.Duration
	PreStopDelay    time.Duration

	RequestTimeout    time.Duration
	RateLimitRPS      int
	RateLimitBurst    int
	MaxBodyBytes      int64
	TrustProxyHeaders bool
}

// Load resolves the configuration from getenv. Every variable is optional:
// an unset or empty one takes its default. All parse failures are reported
// together, each naming its variable and the value received.
func Load(getenv func(string) string) (Config, error) {
	if getenv == nil {
		return Config{}, errors.New("config: getenv is nil")
	}
	l := loader{getenv: getenv}

	cfg := Config{
		Host: l.text("HOST", DefaultHost),
		Port: l.integer("PORT", DefaultPort, 0, maxPort),

		LogLevel:  l.level("LOG_LEVEL", DefaultLogLevel),
		LogFormat: l.format("LOG_FORMAT", DefaultLogFormat),

		CORSAllowedOrigins: l.csv("CORS_ALLOWED_ORIGINS"),

		ReadHeaderTimeout: l.duration("READ_HEADER_TIMEOUT", DefaultReadHeaderTimeout),
		ReadTimeout:       l.duration("READ_TIMEOUT", DefaultReadTimeout),
		WriteTimeout:      l.duration("WRITE_TIMEOUT", DefaultWriteTimeout),
		IdleTimeout:       l.duration("IDLE_TIMEOUT", DefaultIdleTimeout),

		ShutdownTimeout: l.positiveDuration("SHUTDOWN_TIMEOUT", DefaultShutdownTimeout),
		PreStopDelay:    l.duration("PRE_STOP_DELAY", DefaultPreStopDelay),

		RequestTimeout:    l.duration("REQUEST_TIMEOUT", DefaultRequestTimeout),
		RateLimitRPS:      l.integer("RATE_LIMIT_RPS", DefaultRateLimitRPS, 1, maxInt),
		RateLimitBurst:    l.integer("RATE_LIMIT_BURST", DefaultRateLimitBurst, 1, maxInt),
		MaxBodyBytes:      l.integer64("MAX_BODY_BYTES", DefaultMaxBodyBytes, 1, maxInt64),
		TrustProxyHeaders: l.boolean("TRUST_PROXY_HEADERS", DefaultTrustProxyHeaders),
	}
	if err := errors.Join(l.errs...); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Addr is the address the server listens on.
func (c Config) Addr() string { return net.JoinHostPort(c.Host, strconv.Itoa(c.Port)) }

// String renders every resolved value for the startup log. Nothing in Config
// is sensitive — there are no credentials, tokens or connection strings — so
// nothing is redacted; a secret added later must be masked here.
func (c Config) String() string {
	return fmt.Sprintf(
		"host=%s port=%d logLevel=%s logFormat=%s corsAllowedOrigins=[%s] "+
			"readHeaderTimeout=%s readTimeout=%s writeTimeout=%s idleTimeout=%s "+
			"shutdownTimeout=%s preStopDelay=%s requestTimeout=%s "+
			"rateLimitRPS=%d rateLimitBurst=%d maxBodyBytes=%d trustProxyHeaders=%t",
		c.Host, c.Port, levelName(c.LogLevel), c.LogFormat, strings.Join(c.CORSAllowedOrigins, ","),
		c.ReadHeaderTimeout, c.ReadTimeout, c.WriteTimeout, c.IdleTimeout,
		c.ShutdownTimeout, c.PreStopDelay, c.RequestTimeout,
		c.RateLimitRPS, c.RateLimitBurst, c.MaxBodyBytes, c.TrustProxyHeaders)
}
