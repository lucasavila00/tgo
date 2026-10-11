// Package main provides the TGo command.
//
// The command owns file output. The compiler returns file contents so callers
// can choose where to write them. This keeps command-line policy out of the
// compiler API.
//
// In-place output keeps generated Go beside its TGo source. A separate output
// directory provides a complete emitted module. Matching file names preserve
// Go's test and platform selection rules. Unchanged files keep their existing
// contents on disk.
//
// Language rules belong to the compiler. Command-line options choose the source
// and destination; they do not change the meaning of a program.
package main
