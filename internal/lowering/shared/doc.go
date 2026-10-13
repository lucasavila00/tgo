// Package shared preserves Go evaluation and storage for TGo lowering rules.
//
// All rules must agree on when operands run, what values they produce, and
// where assignments store them. Result keeps required statements with typed
// values. The caller places those statements before it uses the values. For
// assignment destinations, shared operations save the address operands without
// reading the destination. This distinction preserves stores and panic order.
//
// Context separates ownership. Input contains checked source facts and selected
// operations, fixed before rewriting starts. Request state owns generated names
// and completion records. File state owns imports and binding changes, including
// names used by closures. Function state owns return values, labels, jumps, and
// saved values. An address operation uses a local context to delay insertion
// until its address checks finish; it does not change the function's state.
//
// Shared operations call the rule dispatcher with the current context when an
// operand contains TGo syntax. A source closure gets new function state. A value
// switch passes its branch transformation explicitly. These boundaries keep
// nested operations in their source function and keep dependencies one-way.
package shared
