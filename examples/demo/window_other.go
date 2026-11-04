//go:build !darwin

package main

import "errors"

func runWindow(*Demo) error {
	return errors.New("-window needs macOS (Windows and Linux windows are not implemented yet)")
}
