//go:build linux
package parity
type record struct {
Short int
Longer int
}
func transform(first int,second int)(int,error){
item:=record{
Short:1, // short
Longer:2, // longer
}
return - -item.Short,nil
}
const(
short=1 // first run
longer=2 // first run
separator=3
left=4 // second run
right=5 // second run
)
type compact struct{Value int}
type documented struct{
Value int

// Keep this comment inside the struct.
}
const top=1 // top comment
var topText string // aligned top comment
var values=[]string{
"short", // first value
"a much longer name", // second value
}
var unicode=[]string{
"plain", // first rune value
"☹", // second rune value
}
var(
reader=make(chan int)
names []string
writer=make(chan int)
)
type inlineComment struct{ // explanation
Value int
}
func compactFunctionWithLongName(value string){println("this body still fits")}
