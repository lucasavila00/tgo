// Package lowering inserts statements into typed Go expressions and lowers error propagation.
//
// The lowerer changes typed Go expressions into statements and result references.
// Its input is a checked package. Load loads one ordinary Go package;
// SourceFromPackage accepts a package already loaded by the driver.
//
// Source.InsertAfter inserts callback statements after a selected expression.
// Source.LowerPropagations replaces recorded propagation calls with typed saved
// results, error checks, and returns. Both operations return formatted Go files.
// Changed AST nodes are copied so the typed source can be used again.
//
// For use(first(), target()!!, last()), lowering saves the earlier call before
// checking target()'s error. An error returns before last() runs. Conditional
// operands stay in their branches. Assignment lowering keeps the original store
// object and the required check timing. Each function has its own return target.
//
// Fresh names and explicit checked types preserve source bindings and conversions.
// See the correctness report for
// implementation rules and the tests that check them.
package lowering
