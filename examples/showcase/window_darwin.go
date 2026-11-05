//go:build darwin

package main

import "github.com/sivaneshwararaj/flexui/platform/darwin"

func runWindow(s *Showcase) error { return darwin.Run(s.App) }
