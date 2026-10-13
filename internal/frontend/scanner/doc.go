// Package scanner reads Go tokens and TGo postfix error operators.
//
// In TGo mode, an adjacent ! or !! after a closing parenthesis is one token.
// The token can end a statement, so newlines, comments, and end of input
// follow Go's semicolon insertion rules. The parser checks whether the
// preceding expression is a call. Go's error-list type keeps diagnostics
// compatible with frontend callers.
//
// scripts/prepare-frontend.sh builds the scanner from the pinned Go source
// and applies the tracked frontend patch.
package scanner
