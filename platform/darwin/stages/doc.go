// Package stages holds the staged verification commands of the macOS
// backend (see ../README.md). Each stage is a separate main package:
//
//	go run ./platform/darwin/stages/stage1 [-debug]
//
// The stages only run on macOS; elsewhere they print a message and exit.
package stages
