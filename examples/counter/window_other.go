//go:build !darwin && !windows

package main

import (
	"errors"

	"github.com/sivaneshwararaj/flexui"
)

func runWindow(*flexui.App) error {
	return errors.New("-window needs macOS or Windows (Linux windows are not implemented yet)")
}
