package main

import "os"

// lintBreak deliberately ignores an error so golangci-lint (errcheck) fails. Throwaway PR for P0-04.
func lintBreak() { os.Remove("does-not-exist") }
