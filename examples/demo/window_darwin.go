//go:build darwin

package main

import "github.com/sivaneshwararaj/flexui/platform/darwin"

func runWindow(d *Demo) error { return darwin.Run(d.App) }
