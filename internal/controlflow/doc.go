// Package controlflow determines whether execution can reach an if branch's value.
//
// An if expression can contain statements before its final value. A return or
// jump can leave the branch without producing that value. For example:
//
//	name := if ready {
//	    if stopped { return }
//	    "ready"
//	} else {
//	    "waiting"
//	}
//
// The first branch supplies "ready" only when stopped is false. The checker
// checks that value's type. Lowering assigns it to the result on those paths. The checker records which branches need a
// value. Lowering uses those checked facts.
//
// Branch reports two facts. Tail says that execution can reach the last
// statement. End says that execution can continue past the branch's end. These
// facts differ for a branch whose last statement is return: that statement can
// be reached, but execution cannot continue past it.
//
// Each analysis builds branch paths, then checks reachability. The builder
// owns mutable label entries because a goto can refer to a later label. Once
// construction is complete, reachability only reads the links.
//
// Paths include loops, switches, and labels. Each analysis stays in its source
// function. The caller supplies the
// rules that syntax alone cannot show, such as which calls terminate execution
// and which expressions contain TGo branch statements.
package controlflow
