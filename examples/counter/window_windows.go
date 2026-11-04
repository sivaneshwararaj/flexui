//go:build windows

package main

import (
	"github.com/sivaneshwararaj/flexui"
	"github.com/sivaneshwararaj/flexui/platform/windows"
)

// runWindow runs the app in a native Windows window.
func runWindow(app *flexui.App) error { return windows.Run(app) }
