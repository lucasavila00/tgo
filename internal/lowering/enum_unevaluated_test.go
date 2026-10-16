package lowering

import "testing"

// TestEnumConstantOperands keeps enum construction out of constant Go operations.
func TestEnumConstantOperands(t *testing.T) {
	input := `package main
import "fmt"
import "unsafe"
type Choice enum { Number struct { Value int }; Callback struct { Run func() } }
const count = len([3]Choice{Choice.Number{Value: 1}})
const size = unsafe.Sizeof(Choice.Number{Value: 1})
const truncated = 1.0 / len([2]Choice{Choice.Number{}})
type Buffer [len([2]Choice{Choice.Number{}})]byte
func main() {
 value := 4
 if truncated != 0 { panic("typed len") }
 const (
 local = len([2]Choice{Choice.Number{Value: value}}) + iota
 inherited
)
 if inherited != 3 { panic("iota") }
 const aligned = unsafe.Alignof(Choice.Number{Value: value})
 var buffer [cap([2]Choice{Choice.Number{Value: value}})]byte
 len, cap, uintptr := 11, 12, 13
 const shadowed = unsafe.Sizeof(Choice.Number{Value: value})
 fmt.Println(count, local, len, cap, uintptr, size == shadowed, aligned > 0, buffer, Buffer{})
}
`
	checkNativeOutput(t, input, "3 2 11 12 13 true true [0 0] [0 0]\n")
}

// TestEnumElidedArrayRange keeps both array ranges and uncalled closures unevaluated.
func TestEnumElidedArrayRange(t *testing.T) {
	input := `package main
import "fmt"
type Choice enum { Number struct { Value int }; Callback struct { Run func() } }
func main() {
 value := 4
 sum := 0
 for i := range [2]Choice{Choice.Number{Value:value}} { sum += i }
 for range [2]Choice{Choice.Callback{Run:func(){ _ = Choice.Number{Value:value}; panic("not called") }}} { sum++ }
 numbers := []int{i for i := range [2]Choice{Choice.Number{Value:value}}}
 calls := 0
 for range [2]Choice{Choice.Number{Value:func()int{calls++;return 5}()}} {}
 fmt.Println(sum, numbers, calls)
}
`
	checkNativeOutput(t, input, "3 [0 1] 1\n")
}

// TestEnumUnsafeGeneric preserves variable-sized operands and Offsetof's selected field.
func TestEnumUnsafeGeneric(t *testing.T) {
	input := `package main
import "fmt"
import "unsafe"
type Choice enum { Number struct { Value int } }
type Bounded enum { Data struct { Bytes [len([2]Choice{Choice.Number{}})]byte } }
func size[T any](value T) uintptr {
 return unsafe.Sizeof(struct{ Value T; Choice Choice }{value, Choice.Number{Value: 3}})
}
func reference[T any](value T) uintptr { return unsafe.Sizeof(struct{ Value T; Choice Choice }{}) }
func main() {
 index := 99
 const count = len([1][2]Choice{{Choice.Number{}}}[index])
 const offset = unsafe.Offsetof(struct{ Prefix byte; Value Choice }{Value:Choice.Number{}}.Value)
 fmt.Println(count, offset > 0, size([3]int{}) == reference([3]int{}), Bounded.Data{}.DataPayload().Bytes)
}
`
	checkNativeOutput(t, input, "2 true true [0 0]\n")
}

// TestEnumUnevaluatedLabels retains label uses at the containing statement, outside generated closures.
func TestEnumUnevaluatedLabels(t *testing.T) {
	input := `package main
import "fmt"
import "unsafe"
type Choice enum { Number struct { Value int } }
func main() {
again:
 const size = unsafe.Sizeof(Choice.Number{Value:if false { goto again; 0 } else { 1 }})
 type Buffer [unsafe.Sizeof(Choice.Number{Value:if false { goto again; 0 } else { 1 }})]byte
 var buffer [unsafe.Sizeof(Choice.Number{Value:if false { goto again; 0 } else { 1 }})]byte
 fmt.Println(size > 0, len(Buffer{}) == len(buffer))
}
`
	checkNativeOutput(t, input, "true true\n")
}

// TestEnumUnevaluatedTypePositions covers types written in function bodies and signatures.
func TestEnumUnevaluatedTypePositions(t *testing.T) {
	input := `package main
import "fmt"
type Choice enum { Number struct { Value int } }
func identity[T any](value T) T { return value }
func sized() [len([2]Choice{Choice.Number{}})]byte { return [2]byte{} }
func main() {
 value := 3
 array := [len([2]Choice{Choice.Number{Value:value}})]byte{}
 fn := func() [len([2]Choice{Choice.Number{Value:value}})]byte { return array }
 boxed := any(array)
 checked := boxed.([len([2]Choice{Choice.Number{Value:value}})]byte)
 fmt.Println(identity[[len([2]Choice{Choice.Number{Value:value}})]byte](checked), fn(), sized())
}
`
	checkNativeOutput(t, input, "[0 0] [0 0] [0 0]\n")
}

// TestEnumArrayEvaluation distinguishes skipped arrays from calls, receives, and value iteration.
func TestEnumArrayEvaluation(t *testing.T) {
	input := `package main
import "fmt"
import "os"
type Choice enum { Number struct { Value int }; Callback struct { Run func() } }
const ignored = len([2]Choice{Choice.Callback{Run:func(){fmt.Fprintln(os.Stderr,"not called")}}})
func main() {
 value := 4
 len, cap := 11, 12
 sum := 0
 for i := range [2]Choice{Choice.Number{Value:value}} { var checked int = i; sum += checked }
 numbers := []int{i for i := range [2]Choice{Choice.Number{Value:value}}}
 channel := make(chan int, 1)
 channel <- 7
 for range [1]Choice{Choice.Number{Value:<-channel}} { sum++ }
 for _, item := range [1]Choice{Choice.Number{Value:value}} { sum += item.NumberPayload().Value }
 fmt.Println(ignored, sum, numbers, len, cap)
}
`
	checkNativeOutput(t, input, "2 6 [0 1] 11 12\n")
}

// TestEnumRangeConstantLabels keeps labels used inside a range operand's constant unsafe call.
func TestEnumRangeConstantLabels(t *testing.T) {
	input := `package main
import "fmt"
import "unsafe"
type Choice enum { Number struct { Value uintptr } }
func main() {
again:
 for range [2]Choice{Choice.Number{Value:unsafe.Sizeof(if false { goto again; 0 } else { 1 })}} {}
 values := []int{i for i := range [2]Choice{
  Choice.Number{Value:unsafe.Sizeof(if false { goto again; 0 } else { 1 })},
 }}
 fmt.Println(values)
}
`
	checkNativeOutput(t, input, "[0 1]\n")
}

// TestEnumInheritedConstants checks each declaration's value when Go repeats its expression with a new iota.
func TestEnumInheritedConstants(t *testing.T) {
	input := `package main
import "fmt"
import "unsafe"
type Choice enum { Number struct { Value int } }
type Count int
type Text string
type Flag bool
const anchor Flag = true
const (
 first, shifted = len([iota+1]Choice{Choice.Number{}}), 1 << len([iota+1]Choice{Choice.Number{}})
 second, secondShifted
)
const (
 bytes = unsafe.Sizeof([iota+1]Choice{Choice.Number{}})
 doubleBytes
)
const (
 typed Count = Count(len([iota+1]Choice{Choice.Number{}}))
 inheritedTyped
)
const (
 word Text = Text(65 + len([iota+1]Choice{Choice.Number{}}))
 nextWord
)
const (
 fraction = 1.0 / float64(len([iota+1]Choice{Choice.Number{}}))
 nextFraction
)
const (
 equal = len([iota+1]Choice{Choice.Number{}}) == iota+1
 nextEqual
)
func main() {
 const Flag = 7
 const (
  inferred = anchor && len([iota+1]Choice{Choice.Number{}}) == 1
  nextInferred
 )
 if !inferred || nextInferred { panic("hidden type") }
 if word != "B" || nextWord != "C" || fraction != 1 || nextFraction != 0.5 || !equal || !nextEqual {
  panic("constant values")
 }
 const (
  local = len([iota+1]Choice{Choice.Number{}})
  next
  explicit = iota + len([iota+1]Choice{Choice.Number{}})
 )
 fmt.Printf("%d %d %d %d %t %T %d %d %d %d\n",first,second,shifted,secondShifted,
  doubleBytes == 2*bytes,inheritedTyped,inheritedTyped,local,next,explicit)
}
`
	checkNativeOutput(t, input, "1 2 2 4 true main.Count 2 1 2 5\n")
}

// TestEnumInheritedConstantTypes keeps iota-dependent type arguments and blank declarations.
func TestEnumInheritedConstantTypes(t *testing.T) {
	input := `package main
import "fmt"
import "reflect"
type Choice enum { Number struct { Value int } }
type Number[T any] int
const (
 first Number[[len([iota+1]Choice{Choice.Number{}})]byte] = 1
 _
 third
)
func main() {
 fmt.Println(reflect.TypeOf(first) == reflect.TypeOf(Number[[1]byte](0)),
  reflect.TypeOf(third) == reflect.TypeOf(Number[[3]byte](0)))
}
`
	checkNativeOutput(t, input, "true true\n")
}

// TestEnumNestedInheritedConstants removes checked copies inside uncalled inherited function bodies.
func TestEnumNestedInheritedConstants(t *testing.T) {
	input := `package main
import "fmt"
type Choice enum { Number struct { Value int } }
func read() (int,error) { panic("not called") }
const (
 first = len([1]func()error{func()error {
  const (
   inner = len([iota+1]Choice{Choice.Number{}})
   next
  )
  _ = inner + next
  _ = read()!!
  return nil
 }})
 second
)
func main() { fmt.Println(first, second) }
`
	checkNativeOutput(t, input, "1 1\n")
}

// TestEnumInheritedConstantLabels keeps outgoing labels without emitting iota outside its declaration.
func TestEnumInheritedConstantLabels(t *testing.T) {
	input := `package main
import "fmt"
import "unsafe"
type Choice enum { Number struct { Value int } }
func main() {
again:
 const (
  first = unsafe.Sizeof(if false { goto again; [iota+1]Choice{Choice.Number{}} } else {
   [iota+1]Choice{Choice.Number{}}
  })
  second
 )
 fmt.Println(second == 2*first)
}
`
	checkNativeOutput(t, input, "true\n")
}
