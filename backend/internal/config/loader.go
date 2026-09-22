package config

import (
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	maxInt   = math.MaxInt
	maxInt64 = math.MaxInt64
)

// A ParseError reports one environment variable that could not be parsed. It
// quotes the value received: no variable read here is sensitive, so echoing
// it back is safe and makes the failure obvious in the logs.
type ParseError struct {
	Name  string // environment variable, e.g. "PORT"
	Value string // value received
	Err   error  // what is wrong with it, e.g. "must be an integer"
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("config: %s=%q: %v", e.Name, e.Value, e.Err)
}

func (e *ParseError) Unwrap() error { return e.Err }

// loader reads variables through getenv and collects every failure, so one
// run reports all the invalid values instead of only the first.
type loader struct {
	getenv func(string) string
	errs   []error
}

// lookup returns the trimmed value of name and whether it was set to
// something. An unset or blank variable means "use the default".
func (l *loader) lookup(name string) (string, bool) {
	v := strings.TrimSpace(l.getenv(name))
	return v, v != ""
}

func (l *loader) fail(name, value string, format string, args ...any) {
	l.errs = append(l.errs, &ParseError{Name: name, Value: value, Err: fmt.Errorf(format, args...)})
}

// text returns the raw value of name, or def when it is unset.
func (l *loader) text(name, def string) string {
	if v, ok := l.lookup(name); ok {
		return v
	}
	return def
}

// csv splits a comma-separated list, trimming spaces around each item and
// dropping empty ones, so "a, ,b," yields ["a" "b"] and "" yields nil.
func (l *loader) csv(name string) []string {
	v, ok := l.lookup(name)
	if !ok {
		return nil
	}
	var out []string
	for item := range strings.SplitSeq(v, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func (l *loader) integer(name string, def, minimum, maximum int) int {
	v, ok := l.lookup(name)
	if !ok {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		l.fail(name, v, "must be an integer")
		return def
	}
	if n < minimum || n > maximum {
		l.fail(name, v, "must be between %d and %d", minimum, maximum)
		return def
	}
	return n
}

func (l *loader) integer64(name string, def, minimum, maximum int64) int64 {
	v, ok := l.lookup(name)
	if !ok {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		l.fail(name, v, "must be an integer")
		return def
	}
	if n < minimum || n > maximum {
		l.fail(name, v, "must be between %d and %d", minimum, maximum)
		return def
	}
	return n
}

func (l *loader) boolean(name string, def bool) bool {
	v, ok := l.lookup(name)
	if !ok {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		l.fail(name, v, "must be true or false")
		return def
	}
	return b
}

// duration accepts any non-negative Go duration; zero means "no timeout"
// wherever net/http reads the value.
func (l *loader) duration(name string, def time.Duration) time.Duration {
	d, raw, ok := l.parseDuration(name, def)
	if !ok {
		return def
	}
	if d < 0 {
		l.fail(name, raw, "must not be negative")
		return def
	}
	return d
}

// positiveDuration rejects zero as well: a drain without a deadline would
// hang the process forever.
func (l *loader) positiveDuration(name string, def time.Duration) time.Duration {
	d, raw, ok := l.parseDuration(name, def)
	if !ok {
		return def
	}
	if d <= 0 {
		l.fail(name, raw, "must be greater than zero")
		return def
	}
	return d
}

// parseDuration returns the parsed value and the raw string it came from,
// so a range check can quote what was received.
func (l *loader) parseDuration(name string, def time.Duration) (d time.Duration, raw string, ok bool) {
	v, set := l.lookup(name)
	if !set {
		return def, v, false
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		l.fail(name, v, "must be a duration such as 750ms, 5s or 2m")
		return def, v, false
	}
	return d, v, true
}

// levels is the accepted LOG_LEVEL set. slog's own parser also accepts
// offsets such as "info+2", which is not a contract we want to support.
var levels = map[string]slog.Level{
	"debug": slog.LevelDebug,
	"info":  slog.LevelInfo,
	"warn":  slog.LevelWarn,
	"error": slog.LevelError,
}

func (l *loader) level(name string, def slog.Level) slog.Level {
	v, ok := l.lookup(name)
	if !ok {
		return def
	}
	if lvl, found := levels[strings.ToLower(v)]; found {
		return lvl
	}
	l.fail(name, v, "must be one of debug, info, warn, error")
	return def
}

func (l *loader) format(name string, def LogFormat) LogFormat {
	v, ok := l.lookup(name)
	if !ok {
		return def
	}
	switch f := LogFormat(strings.ToLower(v)); f {
	case FormatJSON, FormatText:
		return f
	default:
		l.fail(name, v, "must be one of %s, %s", FormatJSON, FormatText)
		return def
	}
}

// levelName renders a level the way LOG_LEVEL spells it.
func levelName(lvl slog.Level) string { return strings.ToLower(lvl.String()) }
