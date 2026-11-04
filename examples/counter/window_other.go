//go:build !darwin

package main

import (
	"errors"

	"github.com/sivaneshwararaj/flexui"
)

func runWindow(*flexui.App) error {
	return errors.New("-window needs macOS (Windows and Linux windows are not implemented yet)")
}
