//go:build darwin

package main

import (
	"github.com/sivaneshwararaj/flexui"
	"github.com/sivaneshwararaj/flexui/platform/darwin"
)

// runWindow runs the app in a native macOS window.
func runWindow(app *flexui.App) error { return darwin.Run(app) }
