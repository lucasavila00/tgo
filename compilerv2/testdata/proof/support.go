package proof

import "fmt"

var Trace []string
var Reached bool

func effect(name string) int                     { Trace = append(Trace, name); return 1 }
func booleanEffect(name string, value bool) bool { Trace = append(Trace, name); return value }
func pair() (int, int)                           { Trace = append(Trace, "pair"); return 1, 2 }
func consume(values ...int)                      { Trace = append(Trace, "consume:"+fmt.Sprint(values)) }
func observe(value bool)                         { Trace = append(Trace, "observe:"+fmt.Sprint(value)) }
func Reset(reached bool)                         { Trace = nil; Reached = reached }
