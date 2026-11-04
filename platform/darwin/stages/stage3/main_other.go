//go:build !darwin

// Command stage3 is a macOS verification stage (see platform/darwin/README.md).
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("stage3 runs on macOS only (see platform/darwin/README.md)")
	os.Exit(1)
}
