// Package frontend lets the compiler read TGo with Go's parser.
//
// The private Go parser reads ! and !! on call expressions. It records the
// exact operand and operator span. After parsing succeeds, the frontend saves
// source call names and builds calls for checking. This keeps generated helper
// names out of error context. Failed attempts reserve no postfix helper names.
//
// Preparation converts if and switch expressions, comprehensions, and short
// returns to temporary Go syntax and records their meaning. A temporary call
// represents a value expression; its branches still belong to the enclosing
// source function. Checking applies TGo rules to these records. Lowering uses
// them to generate ordinary Go statements.
//
// # Keep source edits and source meaning separate
//
// Each preparation rule returns an edit and its TGo record. One file preparation
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
