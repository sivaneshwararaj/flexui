//go:build !darwin && !windows

package main

import "errors"

func runWindow(*Demo) error {
	return errors.New("-window needs macOS or Windows (Linux windows are not implemented yet)")
}
