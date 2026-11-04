//go:build !windows

// Command stage1 is a Windows verification stage (see platform/windows/README.md).
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("stage1 runs on Windows only (see platform/windows/README.md)")
	os.Exit(1)
}
