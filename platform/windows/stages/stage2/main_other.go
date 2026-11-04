//go:build !windows

// Command stage2 is a Windows verification stage (see platform/windows/README.md).
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("stage2 runs on Windows only (see platform/windows/README.md)")
	os.Exit(1)
}
