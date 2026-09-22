// Package config turns the process environment into a typed, validated
// Config with defaults (12-factor: the environment is the only input).
//
// [Load] takes a getenv function instead of reading os.Environ directly, so
// tests configure a process without touching the real environment. An unset
// or empty variable falls back to its default; an unparsable one is an error
// naming the variable and the value received.
package config
