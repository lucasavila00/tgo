package lowering

import "testing"

// TestConstantFunctionBodies keeps imports and lowers uncalled bodies in explicit and inherited constants.
func TestConstantFunctionBodies(t *testing.T) {
	input := `package main
import "fmt"
import "unsafe"
const single = unsafe.Sizeof(func() (int, error) {
 fmt.Print("not called")
 return 1, ...
})
const (
 first = unsafe.Sizeof(func() (int, error) {
  fmt.Print("not called")
  return 2, ...
 })
 second
)
func main() {
 const (
  local = unsafe.Sizeof(func() (int, error) {
   fmt.Print("not called")
   return 3, ...
  })
  inherited
 )
 if single != first || first != second || local != inherited { panic("constant changed") }
}
`
	checkNativeOutput(t, input, "")
}

// TestConstantNativeOperands keeps direct value expressions unevaluated and retains outside label uses.
func TestConstantNativeOperands(t *testing.T) {
	input := `package main
import "unsafe"
const first = unsafe.Sizeof(if true { panic("not evaluated"); 1 } else { 2 })
func main() {
again:
 const (
  local = unsafe.Sizeof(if false { goto again; [iota+1]int{} } else { [iota+1]int{} })
  inherited
 )
 if first == 0 || inherited != 2*local { panic("constant changed") }
}
`
	checkNativeOutput(t, input, "")
}
