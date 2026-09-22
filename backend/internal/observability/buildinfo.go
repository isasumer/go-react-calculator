package observability

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
)

// Build identity, stamped at link time by `make build`:
//
//	-ldflags "-X github.com/isasumer/go-react-calculator/backend/internal/observability.Version=v1.2.3 …"
//
// Left empty (a plain `go build` or `go run`), [Build] falls back to the
// build information the toolchain embeds — the module version and the VCS
// stamps — and finally to [Unknown].
var (
	Version   string
	Commit    string
	BuildDate string
)

// Unknown is what a field reports when neither the linker nor the embedded
// build information supplied it.
const Unknown = "dev"

// shortCommitLen matches `git rev-parse --short`, so a stamped binary and an
// unstamped one report the commit the same way.
const shortCommitLen = 7

// BuildInfo is the resolved identity of the running binary and the body of
// GET /version.
type BuildInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"buildDate"`
	GoVersion string `json:"goVersion"`
}

// Build resolves this binary's build identity.
func Build() BuildInfo {
	info, ok := debug.ReadBuildInfo()
	return resolve(Version, Commit, BuildDate, info, ok)
}

// String renders the identity on one line, for the -version flag and the
// startup log.
func (b BuildInfo) String() string {
	return fmt.Sprintf("version %s (commit %s, built %s, %s)",
		b.Version, b.Commit, b.BuildDate, b.GoVersion)
}

// resolve layers the three sources: link-time values win, then the embedded
// build information, then [Unknown]. It takes info as an argument so the
// fallback is testable without building a binary.
func resolve(version, commit, buildDate string, info *debug.BuildInfo, ok bool) BuildInfo {
	b := BuildInfo{
		Version:   version,
		Commit:    commit,
		BuildDate: buildDate,
		GoVersion: runtime.Version(),
	}
	if !ok || info == nil {
		return b.withDefaults()
	}

	if b.Version == "" && info.Main.Version != "" && info.Main.Version != "(devel)" {
		b.Version = info.Main.Version
	}
	if info.GoVersion != "" {
		b.GoVersion = info.GoVersion
	}

	var revision, modified string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			modified = s.Value
		case "vcs.time":
			if b.BuildDate == "" {
				b.BuildDate = s.Value
			}
		}
	}
	if b.Commit == "" && revision != "" {
		b.Commit = revision
		if len(b.Commit) > shortCommitLen {
			b.Commit = b.Commit[:shortCommitLen]
		}
		if modified == "true" {
			b.Commit += "-dirty"
		}
	}
	return b.withDefaults()
}

// withDefaults fills whatever no source supplied.
func (b BuildInfo) withDefaults() BuildInfo {
	if strings.TrimSpace(b.Version) == "" {
		b.Version = Unknown
	}
	if strings.TrimSpace(b.Commit) == "" {
		b.Commit = Unknown
	}
	if strings.TrimSpace(b.BuildDate) == "" {
		b.BuildDate = Unknown
	}
	return b
}
