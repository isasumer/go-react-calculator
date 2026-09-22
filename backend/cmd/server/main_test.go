package main

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestRun_StartsServesAndShutsDownOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	pr, pw := io.Pipe()
	done := make(chan error, 1)
	go func() {
		done <- run(ctx, []string{"server", "-addr", "127.0.0.1:0"}, os.Getenv, pw)
		_ = pw.Close()
	}()

	// The first log line announces the bound address.
	lines := bufio.NewScanner(pr)
	if !lines.Scan() {
		t.Fatalf("no log output: %v", lines.Err())
	}
	var entry struct {
		Msg  string `json:"msg"`
		Addr string `json:"addr"`
	}
	if err := json.Unmarshal(lines.Bytes(), &entry); err != nil {
		t.Fatalf("decode log line %q: %v", lines.Text(), err)
	}
	if entry.Msg != "listening" || entry.Addr == "" {
		t.Fatalf("unexpected first log line: %s", lines.Text())
	}
	go func() { _, _ = io.Copy(io.Discard, pr) }()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+entry.Addr+"/anything", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run returned %v, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("run did not return after context cancellation")
	}
}

func TestRun_Errors(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "unknown flag", args: []string{"server", "-nope"}},
		{name: "invalid address", args: []string{"server", "-addr", "not-an-address"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := run(t.Context(), tt.args, os.Getenv, io.Discard); err == nil {
				t.Fatal("run returned nil, want error")
			}
		})
	}
}
