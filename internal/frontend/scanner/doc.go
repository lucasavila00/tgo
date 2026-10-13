// Package scanner reads Go tokens and TGo postfix error operators.
//
// In TGo mode, ! or !! after a closing parenthesis is one token. Spaces and
// comments can precede the suffix; the two bytes of !! must be adjacent.
// A suffix can end a statement. Other tokens retain Go's meaning.
//
// Semicolon insertion follows the source tokens, including newlines in
// comments. Buffered lookahead scans each token once and retains its exact
// span. The parser can inspect future tokens without changing the end of
// the current token. Physical file identity stays separate from the logical
// source name used to resolve line directives.
//
// scripts/prepare-frontend.sh builds the scanner from the shared pinned Go
// source and applies the tracked frontend patch.
package scanner
