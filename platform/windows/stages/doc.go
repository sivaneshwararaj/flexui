// Package stages holds the staged verification commands of the Windows
// backend (see ../README.md). Each stage is a separate main package:
//
//	go run ./platform/windows/stages/stage1 [-debug]
//
// The stages only run on Windows; elsewhere they print a message and exit.
package stages
