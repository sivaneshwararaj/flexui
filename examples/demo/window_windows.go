//go:build windows

package main

import "github.com/sivaneshwararaj/flexui/platform/windows"

func runWindow(d *Demo) error { return windows.Run(d.App) }
