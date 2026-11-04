//go:build !darwin

// Command stage4 is a macOS verification stage (see platform/darwin/README.md).
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("stage4 runs on macOS only (see platform/darwin/README.md)")
	os.Exit(1)
}
