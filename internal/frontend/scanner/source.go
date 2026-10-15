package scanner

import (
	"go/token"
	"path/filepath"
)

// InitSource keeps file's physical identity and resolves line directives from sourceName.
// The caller sets the file's initial logical position before scanning.
func (s *Scanner) InitSource(file *token.File, sourceName string, input []byte, handler ErrorHandler, mode Mode) {
	s.Init(file, input, handler, mode)
	s.dir, _ = filepath.Split(sourceName)
}
