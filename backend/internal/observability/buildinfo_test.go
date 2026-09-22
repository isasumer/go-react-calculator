package observability

import (
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
)

// stamped is the build information the Go toolchain embeds for a binary
// built from a clean checkout of a tagged module.
func stamped() *debug.BuildInfo {
	return &debug.BuildInfo{
		GoVersion: "go1.27.1",
		Main:      debug.Module{Path: "example.com/m", Version: "v1.4.0"},
		Settings: []debug.BuildSetting{
			{Key: "vcs", Value: "git"},
			{Key: "vcs.revision", Value: "0123456789abcdef0123456789abcdef01234567"},
			{Key: "vcs.time", Value: "2026-09-22T10:11:12Z"},
			{Key: "vcs.modified", Value: "false"},
		},
	}
}

func TestResolve(t *testing.T) {
	dirty := stamped()
	dirty.Settings[3].Value = "true"

	devel := stamped()
	devel.Main.Version = "(devel)"

	bare := &debug.BuildInfo{GoVersion: "go1.27.1", Main: debug.Module{Path: "example.com/m"}}

	tests := []struct {
		name                  string
		version, commit, date string
		info                  *debug.BuildInfo
		ok                    bool
		want                  BuildInfo
	}{
		{
			name:    "link-time values win",
			version: "v2.0.0", commit: "abc1234", date: "2026-01-02T03:04:05Z",
			info: stamped(), ok: true,
			want: BuildInfo{Version: "v2.0.0", Commit: "abc1234", BuildDate: "2026-01-02T03:04:05Z", GoVersion: "go1.27.1"},
		},
		{
			name: "falls back to embedded build info",
			info: stamped(), ok: true,
			want: BuildInfo{Version: "v1.4.0", Commit: "0123456", BuildDate: "2026-09-22T10:11:12Z", GoVersion: "go1.27.1"},
		},
		{
			name: "dirty working tree is marked",
			info: dirty, ok: true,
			want: BuildInfo{Version: "v1.4.0", Commit: "0123456-dirty", BuildDate: "2026-09-22T10:11:12Z", GoVersion: "go1.27.1"},
		},
		{
			name: "(devel) module version is not a version",
			info: devel, ok: true,
			want: BuildInfo{Version: Unknown, Commit: "0123456", BuildDate: "2026-09-22T10:11:12Z", GoVersion: "go1.27.1"},
		},
		{
			name: "build info without vcs stamps",
			info: bare, ok: true,
			want: BuildInfo{Version: Unknown, Commit: Unknown, BuildDate: Unknown, GoVersion: "go1.27.1"},
		},
		{
			name: "no build info at all",
			info: nil, ok: false,
			want: BuildInfo{Version: Unknown, Commit: Unknown, BuildDate: Unknown, GoVersion: runtime.Version()},
		},
		{
			name: "nil build info reported as present",
			info: nil, ok: true,
			want: BuildInfo{Version: Unknown, Commit: Unknown, BuildDate: Unknown, GoVersion: runtime.Version()},
		},
		{
			name:   "only the commit is stamped",
			commit: "deadbee",
			info:   bare, ok: true,
			want: BuildInfo{Version: Unknown, Commit: "deadbee", BuildDate: Unknown, GoVersion: "go1.27.1"},
		},
		{
			name:    "blank link-time values fall through",
			version: "  ", commit: " ", date: "\t",
			info: nil, ok: false,
			want: BuildInfo{Version: Unknown, Commit: Unknown, BuildDate: Unknown, GoVersion: runtime.Version()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolve(tt.version, tt.commit, tt.date, tt.info, tt.ok); got != tt.want {
				t.Errorf("resolve() = %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

// TestBuild covers the package-level variables the linker writes to.
func TestBuild(t *testing.T) {
	t.Run("unstamped test binary", func(t *testing.T) {
		got := Build()
		if got.Version == "" || got.Commit == "" || got.BuildDate == "" {
			t.Errorf("Build() = %+v, want every field filled", got)
		}
		if got.GoVersion == "" {
			t.Errorf("Build().GoVersion is empty")
		}
	})

	t.Run("linker values", func(t *testing.T) {
		// The linker sets these; a test may too, as long as it restores them.
		defer func(v, c, d string) { Version, Commit, BuildDate = v, c, d }(Version, Commit, BuildDate)
		Version, Commit, BuildDate = "v9.9.9", "cafebab", "2026-09-22T00:00:00Z"

		got := Build()
		if got.Version != "v9.9.9" || got.Commit != "cafebab" || got.BuildDate != "2026-09-22T00:00:00Z" {
			t.Errorf("Build() = %+v, want the link-time values", got)
		}
	})
}

func TestBuildInfo_String(t *testing.T) {
	b := BuildInfo{Version: "v1.0.0", Commit: "abc1234", BuildDate: "2026-09-22T10:11:12Z", GoVersion: "go1.27.1"}
	got := b.String()
	for _, want := range []string{"v1.0.0", "abc1234", "2026-09-22T10:11:12Z", "go1.27.1"} {
		if !strings.Contains(got, want) {
			t.Errorf("String() = %q, missing %q", got, want)
		}
	}
}
