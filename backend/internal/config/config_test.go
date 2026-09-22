package config

import (
	"errors"
	"log/slog"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

// env turns a map into a getenv function; a missing key reads as unset.
func env(vars map[string]string) func(string) string {
	return func(name string) string { return vars[name] }
}

// defaults is what an empty environment must resolve to. It is written out
// by hand rather than built from the Default* constants, so a change to a
// default has to be made here too and is never silently accepted.
func defaults() Config {
	return Config{
		Host:               "0.0.0.0",
		Port:               8081,
		LogLevel:           slog.LevelInfo,
		LogFormat:          FormatJSON,
		CORSAllowedOrigins: nil,
		ReadHeaderTimeout:  5 * time.Second,
		ReadTimeout:        10 * time.Second,
		WriteTimeout:       10 * time.Second,
		IdleTimeout:        60 * time.Second,
		ShutdownTimeout:    15 * time.Second,
		PreStopDelay:       0,
		RequestTimeout:     5 * time.Second,
		RateLimitRPS:       20,
		RateLimitBurst:     40,
		MaxBodyBytes:       4096,
		TrustProxyHeaders:  false,
	}
}

func TestLoad_Defaults(t *testing.T) {
	t.Run("unset", func(t *testing.T) {
		got, err := Load(env(nil))
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if want := defaults(); !reflect.DeepEqual(got, want) {
			t.Errorf("Load() = %+v\nwant %+v", got, want)
		}
	})

	// An empty or blank variable is the same as an unset one: a deployment
	// that passes PORT="" gets the default, not a parse error.
	t.Run("blank", func(t *testing.T) {
		blank := map[string]string{}
		for _, name := range []string{
			"HOST", "PORT", "LOG_LEVEL", "LOG_FORMAT", "CORS_ALLOWED_ORIGINS",
			"READ_HEADER_TIMEOUT", "READ_TIMEOUT", "WRITE_TIMEOUT", "IDLE_TIMEOUT",
			"SHUTDOWN_TIMEOUT", "PRE_STOP_DELAY", "REQUEST_TIMEOUT",
			"RATE_LIMIT_RPS", "RATE_LIMIT_BURST", "MAX_BODY_BYTES", "TRUST_PROXY_HEADERS",
		} {
			blank[name] = "   "
		}
		got, err := Load(env(blank))
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if want := defaults(); !reflect.DeepEqual(got, want) {
			t.Errorf("Load() = %+v\nwant %+v", got, want)
		}
	})
}

// TestLoad_Overrides sets one variable at a time, so a variable read under
// the wrong name fails on its own row.
func TestLoad_Overrides(t *testing.T) {
	tests := []struct {
		name  string
		value string
		got   func(Config) any
		want  any
	}{
		{"HOST", "127.0.0.1", func(c Config) any { return c.Host }, "127.0.0.1"},
		{"PORT", "0", func(c Config) any { return c.Port }, 0},
		{"PORT", "9000", func(c Config) any { return c.Port }, 9000},
		{"PORT", "65535", func(c Config) any { return c.Port }, 65535},
		{"LOG_LEVEL", "debug", func(c Config) any { return c.LogLevel }, slog.LevelDebug},
		{"LOG_LEVEL", "info", func(c Config) any { return c.LogLevel }, slog.LevelInfo},
		{"LOG_LEVEL", "warn", func(c Config) any { return c.LogLevel }, slog.LevelWarn},
		{"LOG_LEVEL", "error", func(c Config) any { return c.LogLevel }, slog.LevelError},
		{"LOG_LEVEL", "WARN", func(c Config) any { return c.LogLevel }, slog.LevelWarn},
		{"LOG_FORMAT", "text", func(c Config) any { return c.LogFormat }, FormatText},
		{"LOG_FORMAT", "JSON", func(c Config) any { return c.LogFormat }, FormatJSON},
		{"READ_HEADER_TIMEOUT", "1s", func(c Config) any { return c.ReadHeaderTimeout }, time.Second},
		{"READ_TIMEOUT", "2s", func(c Config) any { return c.ReadTimeout }, 2 * time.Second},
		{"WRITE_TIMEOUT", "3s", func(c Config) any { return c.WriteTimeout }, 3 * time.Second},
		{"IDLE_TIMEOUT", "4m", func(c Config) any { return c.IdleTimeout }, 4 * time.Minute},
		{"IDLE_TIMEOUT", "0s", func(c Config) any { return c.IdleTimeout }, time.Duration(0)},
		{"SHUTDOWN_TIMEOUT", "30s", func(c Config) any { return c.ShutdownTimeout }, 30 * time.Second},
		{"PRE_STOP_DELAY", "250ms", func(c Config) any { return c.PreStopDelay }, 250 * time.Millisecond},
		{"REQUEST_TIMEOUT", "1500ms", func(c Config) any { return c.RequestTimeout }, 1500 * time.Millisecond},
		{"RATE_LIMIT_RPS", "100", func(c Config) any { return c.RateLimitRPS }, 100},
		{"RATE_LIMIT_BURST", "1", func(c Config) any { return c.RateLimitBurst }, 1},
		{"MAX_BODY_BYTES", "65536", func(c Config) any { return c.MaxBodyBytes }, int64(65536)},
		{"TRUST_PROXY_HEADERS", "true", func(c Config) any { return c.TrustProxyHeaders }, true},
		{"TRUST_PROXY_HEADERS", "1", func(c Config) any { return c.TrustProxyHeaders }, true},
		{"TRUST_PROXY_HEADERS", "false", func(c Config) any { return c.TrustProxyHeaders }, false},
		{
			"CORS_ALLOWED_ORIGINS", "http://localhost:5173",
			func(c Config) any { return c.CORSAllowedOrigins },
			[]string{"http://localhost:5173"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name+"="+tt.value, func(t *testing.T) {
			cfg, err := Load(env(map[string]string{tt.name: tt.value}))
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if got := tt.got(cfg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%s = %v (%T), want %v (%T)", tt.name, got, got, tt.want, tt.want)
			}
		})
	}
}

// TestLoad_EveryVariableOverridden also proves no variable is read twice
// under two names: every field differs from its default.
func TestLoad_EveryVariableOverridden(t *testing.T) {
	got, err := Load(env(map[string]string{
		"HOST":                 "127.0.0.1",
		"PORT":                 "9999",
		"LOG_LEVEL":            "debug",
		"LOG_FORMAT":           "text",
		"CORS_ALLOWED_ORIGINS": "http://localhost:5173,https://app.example.com",
		"READ_HEADER_TIMEOUT":  "1s",
		"READ_TIMEOUT":         "2s",
		"WRITE_TIMEOUT":        "3s",
		"IDLE_TIMEOUT":         "4s",
		"SHUTDOWN_TIMEOUT":     "5s",
		"PRE_STOP_DELAY":       "6s",
		"REQUEST_TIMEOUT":      "7s",
		"RATE_LIMIT_RPS":       "8",
		"RATE_LIMIT_BURST":     "9",
		"MAX_BODY_BYTES":       "10",
		"TRUST_PROXY_HEADERS":  "true",
	}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := Config{
		Host:               "127.0.0.1",
		Port:               9999,
		LogLevel:           slog.LevelDebug,
		LogFormat:          FormatText,
		CORSAllowedOrigins: []string{"http://localhost:5173", "https://app.example.com"},
		ReadHeaderTimeout:  time.Second,
		ReadTimeout:        2 * time.Second,
		WriteTimeout:       3 * time.Second,
		IdleTimeout:        4 * time.Second,
		ShutdownTimeout:    5 * time.Second,
		PreStopDelay:       6 * time.Second,
		RequestTimeout:     7 * time.Second,
		RateLimitRPS:       8,
		RateLimitBurst:     9,
		MaxBodyBytes:       10,
		TrustProxyHeaders:  true,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Load() = %+v\nwant %+v", got, want)
	}
}

func TestLoad_CORSAllowedOrigins(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  []string
	}{
		{"unset", "", nil},
		{"single", "http://a.test", []string{"http://a.test"}},
		{"two", "http://a.test,http://b.test", []string{"http://a.test", "http://b.test"}},
		{"spaces", " http://a.test , http://b.test ", []string{"http://a.test", "http://b.test"}},
		{"empty items", "http://a.test,,http://b.test,", []string{"http://a.test", "http://b.test"}},
		{"only separators", ", ,,", nil},
		{"tabs and newlines", "http://a.test,\n\thttp://b.test", []string{"http://a.test", "http://b.test"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Load(env(map[string]string{"CORS_ALLOWED_ORIGINS": tt.value}))
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if !reflect.DeepEqual(cfg.CORSAllowedOrigins, tt.want) {
				t.Errorf("CORSAllowedOrigins = %#v, want %#v", cfg.CORSAllowedOrigins, tt.want)
			}
		})
	}
}

// TestLoad_Invalid checks every rejected value: the error names the variable,
// quotes what it received and says what was expected.
func TestLoad_Invalid(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		reason string
	}{
		{"PORT", "abc", "must be an integer"},
		{"PORT", "8081.0", "must be an integer"},
		{"PORT", "-1", "must be between 0 and 65535"},
		{"PORT", "70000", "must be between 0 and 65535"},
		{"LOG_LEVEL", "verbose", "must be one of debug, info, warn, error"},
		{"LOG_LEVEL", "info+2", "must be one of debug, info, warn, error"},
		{"LOG_FORMAT", "xml", "must be one of json, text"},
		{"READ_HEADER_TIMEOUT", "5", "must be a duration"},
		{"READ_HEADER_TIMEOUT", "-1s", "must not be negative"},
		{"READ_TIMEOUT", "ten seconds", "must be a duration"},
		{"READ_TIMEOUT", "-1ms", "must not be negative"},
		{"WRITE_TIMEOUT", "10 s", "must be a duration"},
		{"WRITE_TIMEOUT", "-10s", "must not be negative"},
		{"IDLE_TIMEOUT", "1 minute", "must be a duration"},
		{"IDLE_TIMEOUT", "-1h", "must not be negative"},
		{"SHUTDOWN_TIMEOUT", "nope", "must be a duration"},
		{"SHUTDOWN_TIMEOUT", "0s", "must be greater than zero"},
		{"SHUTDOWN_TIMEOUT", "-5s", "must be greater than zero"},
		{"PRE_STOP_DELAY", "soon", "must be a duration"},
		{"PRE_STOP_DELAY", "-1s", "must not be negative"},
		{"REQUEST_TIMEOUT", "5sec", "must be a duration"},
		{"REQUEST_TIMEOUT", "-5s", "must not be negative"},
		{"RATE_LIMIT_RPS", "fast", "must be an integer"},
		{"RATE_LIMIT_RPS", "0", "must be between 1 and"},
		{"RATE_LIMIT_RPS", "-20", "must be between 1 and"},
		{"RATE_LIMIT_BURST", "1e3", "must be an integer"},
		{"RATE_LIMIT_BURST", "0", "must be between 1 and"},
		{"MAX_BODY_BYTES", "4k", "must be an integer"},
		{"MAX_BODY_BYTES", "0", "must be between 1 and"},
		{"MAX_BODY_BYTES", "-4096", "must be between 1 and"},
		{"TRUST_PROXY_HEADERS", "yes", "must be true or false"},
		{"TRUST_PROXY_HEADERS", "2", "must be true or false"},
	}
	for _, tt := range tests {
		t.Run(tt.name+"="+tt.value, func(t *testing.T) {
			cfg, err := Load(env(map[string]string{tt.name: tt.value}))
			if err == nil {
				t.Fatalf("Load(%s=%q) = %+v, want an error", tt.name, tt.value, cfg)
			}
			if !reflect.DeepEqual(cfg, Config{}) {
				t.Errorf("Load returned %+v with an error, want the zero Config", cfg)
			}
			msg := err.Error()
			if !strings.Contains(msg, tt.name) {
				t.Errorf("error %q does not name %s", msg, tt.name)
			}
			if !strings.Contains(msg, strconv.Quote(tt.value)) {
				t.Errorf("error %q does not quote the value %q", msg, tt.value)
			}
			if !strings.Contains(msg, tt.reason) {
				t.Errorf("error %q does not say %q", msg, tt.reason)
			}
			perr, ok := errors.AsType[*ParseError](err)
			if !ok {
				t.Fatalf("error %v is not a *ParseError", err)
			}
			if perr.Name != tt.name || perr.Value != tt.value {
				t.Errorf("ParseError = {Name:%q Value:%q}, want {Name:%q Value:%q}",
					perr.Name, perr.Value, tt.name, tt.value)
			}
			if perr.Unwrap() == nil {
				t.Error("ParseError.Unwrap() = nil, want the reason")
			}
		})
	}
}

// TestLoad_ReportsEveryInvalidValue: a misconfigured deployment sees all its
// mistakes at once instead of one per restart.
func TestLoad_ReportsEveryInvalidValue(t *testing.T) {
	_, err := Load(env(map[string]string{
		"PORT":       "abc",
		"LOG_LEVEL":  "verbose",
		"LOG_FORMAT": "xml",
	}))
	if err == nil {
		t.Fatal("Load returned nil, want an error")
	}
	for _, name := range []string{"PORT", "LOG_LEVEL", "LOG_FORMAT"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q does not name %s", err, name)
		}
	}
}

func TestLoad_NilGetenv(t *testing.T) {
	if _, err := Load(nil); err == nil {
		t.Fatal("Load(nil) returned nil, want an error")
	}
}

func TestConfig_Addr(t *testing.T) {
	tests := []struct {
		host string
		port int
		want string
	}{
		{"0.0.0.0", 8081, "0.0.0.0:8081"},
		{"127.0.0.1", 0, "127.0.0.1:0"},
		{"::1", 9000, "[::1]:9000"},
		{"", 8081, ":8081"},
	}
	for _, tt := range tests {
		cfg := Config{Host: tt.host, Port: tt.port}
		if got := cfg.Addr(); got != tt.want {
			t.Errorf("Config{%q, %d}.Addr() = %q, want %q", tt.host, tt.port, got, tt.want)
		}
	}
}

// TestConfig_String keeps the startup log line complete: every variable the
// process resolved has to show up in it.
func TestConfig_String(t *testing.T) {
	cfg, err := Load(env(map[string]string{
		"CORS_ALLOWED_ORIGINS": "http://a.test,http://b.test",
		"LOG_LEVEL":            "warn",
		"LOG_FORMAT":           "text",
	}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := cfg.String()
	want := []string{
		"host=0.0.0.0", "port=8081", "logLevel=warn", "logFormat=text",
		"corsAllowedOrigins=[http://a.test,http://b.test]",
		"readHeaderTimeout=5s", "readTimeout=10s", "writeTimeout=10s", "idleTimeout=1m0s",
		"shutdownTimeout=15s", "preStopDelay=0s", "requestTimeout=5s",
		"rateLimitRPS=20", "rateLimitBurst=40", "maxBodyBytes=4096", "trustProxyHeaders=false",
	}
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("String() = %q\nis missing %q", got, w)
		}
	}
}

func TestLevelName(t *testing.T) {
	tests := map[slog.Level]string{
		slog.LevelDebug: "debug",
		slog.LevelInfo:  "info",
		slog.LevelWarn:  "warn",
		slog.LevelError: "error",
	}
	for lvl, want := range tests {
		if got := levelName(lvl); got != want {
			t.Errorf("levelName(%v) = %q, want %q", lvl, got, want)
		}
	}
}
