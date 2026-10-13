// Package frontend lets the compiler read TGo with Go's parser.
//
// TGo adds syntax such as fetch()! and if expressions. The frontend replaces
// these forms with temporary Go syntax and records their meaning. For example,
// a temporary call represents an if expression; its branches still belong to
// the enclosing source function. The checker uses that record to apply TGo
// rules. Lowering uses it to generate ordinary Go statements.
//
// # Keep source edits and source meaning separate
//
// Each syntax rule returns an edit and its TGo record. One file preparation
// owns the prepared text, records, and positions. It applies text edits and
// position changes together. Errors and syntax nodes thus keep the positions
// of the input, even when a replacement has a different length.
//
// Conditional syntax can be ambiguous with an ordinary Go statement. Its
// preparation and parser recovery share recognition and replacement rules.
// Preparation handles clear operands first. If Go parsing fails, recovery can
// identify another operand and request the same kind of edit. This preserves
// ordinary Go statement parsing and allows nested TGo forms.
//
// ParseFile returns the completed tree and TGo records together. Type checking
// and code generation use that result; source preparation does not choose types.
package frontend
