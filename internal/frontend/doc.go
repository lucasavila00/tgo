// Package frontend reads TGo syntax into a Go syntax tree.
//
// ParseFile accepts source bytes and returns an AST with operation records.
// Go's standard parser reads headers and statements. The frontend prepares
// added syntax; the parser itself is not patched.
//
// For postfix propagation, a named comment lets Go parse the surrounding call.
// The frontend then replaces that call with a marker node and records the
// operator, call name, and source position.
//
// For if values, it prepares clear expression positions before parsing the
// file. Parsed syntax resolves ambiguous positions, such as colons in labels,
// map values, and slice bounds. Independent values share preparation edits;
// nested values can need more rounds. Placeholder function literals carry
// branch statements to the checker without creating source function boundaries.
//
// Offsets map prepared text to original source bytes. Physical generated paths
// control build selection; diagnostic positions identify the original TGo file.
// This package does not check types or choose where generated statements run.
//
// See the [checker] for marker types and scope, the [driver] for compilation and
// shared terms, and [measurements] for parsing costs.
//
// [checker]: https://github.com/lucasavila00/tgo/blob/main/internal/checker/readme.md
// [driver]: https://github.com/lucasavila00/tgo/blob/main/internal/driver/readme.md
// [measurements]: https://github.com/lucasavila00/tgo/blob/main/docs/performance.md
package frontend
