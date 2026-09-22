// Package api carries the machine-readable contract of the service.
//
// openapi.yaml lives here rather than next to the handler because it is the
// contract, not an implementation detail: the frontend, the docs and the
// contract tests all read this one file. It is embedded, so the binary serves
// the exact bytes it was built from and a deployed instance cannot describe
// an API it does not implement.
package api

import (
	"embed"
	"fmt"
)

// SpecPath is the name of the contract inside [SpecFS], and the last segment
// of the URL the server publishes it at.
const SpecPath = "openapi.yaml"

//go:embed openapi.yaml
var specFS embed.FS

// Spec returns the contract's bytes. The copy is read once at package
// initialisation, so serving it is a slice header and no I/O.
func Spec() []byte { return spec }

var spec = mustReadSpec()

// mustReadSpec reads the embedded file at initialisation. The read cannot
// fail — the bytes are part of the binary and //go:embed would not have
// compiled without them — so a failure here is a broken build, not a runtime
// condition a caller could handle.
func mustReadSpec() []byte {
	b, err := specFS.ReadFile(SpecPath)
	if err != nil {
		panic(fmt.Sprintf("api: embedded %s: %v", SpecPath, err))
	}
	return b
}
