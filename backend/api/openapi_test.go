package api

import (
	"bytes"
	"os"
	"testing"
)

// TestSpecIsTheFileOnDisk pins the one property this package has: what the
// binary serves is openapi.yaml as it sits in the repository. Nothing
// generates or rewrites the contract at build time, so reviewing the file is
// reviewing what a client will fetch.
//
// What the document *says* is asserted where it can be held against the
// implementation: internal/httpapi/openapi_test.go validates it and every
// response the handler produces against it.
func TestSpecIsTheFileOnDisk(t *testing.T) {
	want, err := os.ReadFile(SpecPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(want) == 0 {
		t.Fatal("openapi.yaml is empty")
	}
	if !bytes.Equal(Spec(), want) {
		t.Errorf("the embedded spec is %d bytes, %s is %d", len(Spec()), SpecPath, len(want))
	}
}
