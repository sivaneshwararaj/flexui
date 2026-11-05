//go:build windows

package main

import "github.com/sivaneshwararaj/flexui/platform/windows"

func runWindow(s *Showcase) error { return windows.Run(s.App) }
