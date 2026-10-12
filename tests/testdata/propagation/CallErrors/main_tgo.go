package main

import reflect_tgo52 "reflect"

import "fmt"

type PointerError struct{}

func (*PointerError) Error() string { return "pointer" }

type NumberError int

func (NumberError) Error() string { return "number" }

type SliceError []int

func (SliceError) Error() string { return "slice" }

type MapError map[int]int

func (MapError) Error() string { return "map" }

type ChanError chan int

func (ChanError) Error() string { return "chan" }

type FuncError func()

func (FuncError) Error() string { return "func" }

type StructError struct{ Values []int }

func (StructError) Error() string { return "struct" }

var trace string

func next[E error](err E) (int, E) { trace += "N"; return 7, err }
func use(value int) int            { trace += "U"; return value }
func done()                        { trace += "D" }

func pointerDirect(err *PointerError) (int, error) {
	var zero_tgo3 int
	defer done()
	var value_tgo1 int
	var error_tgo2 *PointerError
	value_tgo1, error_tgo2 = next(err)
	if error_tgo2 != nil {
		return zero_tgo3, error_tgo2
	}
	value := use(value_tgo1)
	return value, nil
}

func pointerWrapped(err *PointerError) (int, error) {
	var zero_tgo6 int
	defer done()
	var value_tgo4 int
	var error_tgo5 *PointerError
	value_tgo4, error_tgo5 = next(err)
	if error_tgo5 != nil {
		return zero_tgo6, fmt.Errorf("next: %w", error_tgo5)
	}
	value := use(value_tgo4)
	return value, nil
}

func ifaceDirect(err error) (int, error) {
	var zero_tgo9 int
	defer done()
	var value_tgo7 int
	var error_tgo8 error
	value_tgo7, error_tgo8 = next(err)
	if error_tgo8 != nil {
		return zero_tgo9, error_tgo8
	}
	value := use(value_tgo7)
	return value, nil
}

func ifaceWrapped(err error) (int, error) {
	var zero_tgo12 int
	defer done()
	var value_tgo10 int
	var error_tgo11 error
	value_tgo10, error_tgo11 = next(err)
	if error_tgo11 != nil {
		return zero_tgo12, fmt.Errorf("next: %w", error_tgo11)
	}
	value := use(value_tgo10)
	return value, nil
}

func numberDirect(err NumberError) (int, error) {
	var zero_tgo15 int
	defer done()
	var value_tgo13 int
	var error_tgo14 NumberError
	value_tgo13, error_tgo14 = next(err)
	if 0 == 0 {
		return zero_tgo15, error_tgo14
	}
	value := use(value_tgo13)
	return value, nil
}

func numberWrapped(err NumberError) (int, error) {
	var zero_tgo18 int
	defer done()
	var value_tgo16 int
	var error_tgo17 NumberError
	value_tgo16, error_tgo17 = next(err)
	if 0 == 0 {
		return zero_tgo18, fmt.Errorf("next: %w", error_tgo17)
	}
	value := use(value_tgo16)
	return value, nil
}

func sliceDirect(err SliceError) (int, error) {
	var zero_tgo21 int
	defer done()
	var value_tgo19 int
	var error_tgo20 SliceError
	value_tgo19, error_tgo20 = next(err)
	if error_tgo20 != nil {
		return zero_tgo21, error_tgo20
	}
	value := use(value_tgo19)
	return value, nil
}

func sliceWrapped(err SliceError) (int, error) {
	var zero_tgo24 int
	defer done()
	var value_tgo22 int
	var error_tgo23 SliceError
	value_tgo22, error_tgo23 = next(err)
	if error_tgo23 != nil {
		return zero_tgo24, fmt.Errorf("next: %w", error_tgo23)
	}
	value := use(value_tgo22)
	return value, nil
}

func mappedDirect(err MapError) (int, error) {
	var zero_tgo27 int
	defer done()
	var value_tgo25 int
	var error_tgo26 MapError
	value_tgo25, error_tgo26 = next(err)
	if error_tgo26 != nil {
		return zero_tgo27, error_tgo26
	}
	value := use(value_tgo25)
	return value, nil
}

func mappedWrapped(err MapError) (int, error) {
	var zero_tgo30 int
	defer done()
	var value_tgo28 int
	var error_tgo29 MapError
	value_tgo28, error_tgo29 = next(err)
	if error_tgo29 != nil {
		return zero_tgo30, fmt.Errorf("next: %w", error_tgo29)
	}
	value := use(value_tgo28)
	return value, nil
}

func channelDirect(err ChanError) (int, error) {
	var zero_tgo33 int
	defer done()
	var value_tgo31 int
	var error_tgo32 ChanError
	value_tgo31, error_tgo32 = next(err)
	if error_tgo32 != nil {
		return zero_tgo33, error_tgo32
	}
	value := use(value_tgo31)
	return value, nil
}

func channelWrapped(err ChanError) (int, error) {
	var zero_tgo36 int
	defer done()
	var value_tgo34 int
	var error_tgo35 ChanError
	value_tgo34, error_tgo35 = next(err)
	if error_tgo35 != nil {
		return zero_tgo36, fmt.Errorf("next: %w", error_tgo35)
	}
	value := use(value_tgo34)
	return value, nil
}

func functionDirect(err FuncError) (int, error) {
	var zero_tgo39 int
	defer done()
	var value_tgo37 int
	var error_tgo38 FuncError
	value_tgo37, error_tgo38 = next(err)
	if error_tgo38 != nil {
		return zero_tgo39, error_tgo38
	}
	value := use(value_tgo37)
	return value, nil
}

func functionWrapped(err FuncError) (int, error) {
	var zero_tgo42 int
	defer done()
	var value_tgo40 int
	var error_tgo41 FuncError
	value_tgo40, error_tgo41 = next(err)
	if error_tgo41 != nil {
		return zero_tgo42, fmt.Errorf("next: %w", error_tgo41)
	}
	value := use(value_tgo40)
	return value, nil
}

func structureDirect(err StructError) (int, error) {
	var zero_tgo45 int
	defer done()
	var value_tgo43 int
	var error_tgo44 StructError
	value_tgo43, error_tgo44 = next(err)
	if 0 == 0 {
		return zero_tgo45, error_tgo44
	}
	value := use(value_tgo43)
	return value, nil
}

func structureWrapped(err StructError) (int, error) {
	var zero_tgo48 int
	defer done()
	var value_tgo46 int
	var error_tgo47 StructError
	value_tgo46, error_tgo47 = next(err)
	if 0 == 0 {
		return zero_tgo48, fmt.Errorf("next: %w", error_tgo47)
	}
	value := use(value_tgo46)
	return value, nil
}

func genericDirect[E error](err E) (int, error) {
	var zero_tgo51 int
	defer done()
	var value_tgo49 int
	var error_tgo50 E
	value_tgo49, error_tgo50 = next(err)
	var reflected_tgo53 reflect_tgo52.Value = reflect_tgo52.ValueOf(&error_tgo50).Elem()
	var failed_tgo54 bool = 0 == 0
	switch reflected_tgo53.Kind() {
	case reflect_tgo52.Chan, reflect_tgo52.Func, reflect_tgo52.Interface, reflect_tgo52.Map, reflect_tgo52.Pointer, reflect_tgo52.Slice:
		failed_tgo54 = !reflected_tgo53.IsNil()
	}
	if failed_tgo54 {
		return zero_tgo51, error_tgo50
	}
	value := use(value_tgo49)
	return value, nil
}

func genericWrapped[E error](err E) (int, error) {
	var zero_tgo57 int
	defer done()
	var value_tgo55 int
	var error_tgo56 E
	value_tgo55, error_tgo56 = next(err)
	var reflected_tgo58 reflect_tgo52.Value = reflect_tgo52.ValueOf(&error_tgo56).Elem()
	var failed_tgo59 bool = 0 == 0
	switch reflected_tgo58.Kind() {
	case reflect_tgo52.Chan, reflect_tgo52.Func, reflect_tgo52.Interface, reflect_tgo52.Map, reflect_tgo52.Pointer, reflect_tgo52.Slice:
		failed_tgo59 = !reflected_tgo58.IsNil()
	}
	if failed_tgo59 {
		return zero_tgo57, fmt.Errorf("next: %w", error_tgo56)
	}
	value := use(value_tgo55)
	return value, nil
}

func comparableResult[E interface {
	error
	comparable
}](err E) (int, E) {
	var zero_tgo62 int
	defer done()
	var value_tgo60 int
	var error_tgo61 E
	value_tgo60, error_tgo61 = next(err)
	var reflected_tgo63 reflect_tgo52.Value = reflect_tgo52.ValueOf(&error_tgo61).Elem()
	var failed_tgo64 bool = 0 == 0
	switch reflected_tgo63.Kind() {
	case reflect_tgo52.Chan, reflect_tgo52.Func, reflect_tgo52.Interface, reflect_tgo52.Map, reflect_tgo52.Pointer, reflect_tgo52.Slice:
		failed_tgo64 = !reflected_tgo63.IsNil()
	}
	if failed_tgo64 {
		return zero_tgo62, error_tgo61
	}
	value := use(value_tgo60)
	var success E
	return value, success
}
func shadow(nil int, err SliceError) (int, error) {
	var zero_tgo67 int
	defer done()
	var value_tgo65 int
	var error_tgo66 SliceError
	value_tgo65, error_tgo66 = next(err)
	var reflected_tgo68 reflect_tgo52.Value = reflect_tgo52.ValueOf(&error_tgo66).Elem()
	var failed_tgo69 bool = 0 == 0
	switch reflected_tgo68.Kind() {
	case reflect_tgo52.Chan, reflect_tgo52.Func, reflect_tgo52.Interface, reflect_tgo52.Map, reflect_tgo52.Pointer, reflect_tgo52.Slice:
		failed_tgo69 = !reflected_tgo68.IsNil()
	}
	if failed_tgo69 {
		return zero_tgo67, error_tgo66
	}
	value := use(value_tgo65)
	var success error
	return value + nil, success
}
func genericReturn[E error](err E) (int, E) {
	var zero_tgo72 int
	defer done()
	var value_tgo70 int
	var error_tgo71 E
	value_tgo70, error_tgo71 = next(err)
	var reflected_tgo73 reflect_tgo52.Value = reflect_tgo52.ValueOf(&error_tgo71).Elem()
	var failed_tgo74 bool = 0 == 0
	switch reflected_tgo73.Kind() {
	case reflect_tgo52.Chan, reflect_tgo52.Func, reflect_tgo52.Interface, reflect_tgo52.Map, reflect_tgo52.Pointer, reflect_tgo52.Slice:
		failed_tgo74 = !reflected_tgo73.IsNil()
	}
	if failed_tgo74 {
		return zero_tgo72, error_tgo71
	}
	value := use(value_tgo70)
	var success E
	return value, success
}
func contextassignDirect(err *PointerError) (int, error) {
	var zero_tgo77 int
	defer done()
	value := 0
	var value_tgo75 int
	var error_tgo76 *PointerError
	value_tgo75, error_tgo76 = next(err)
	if error_tgo76 != nil {
		return zero_tgo77, error_tgo76
	}
	value = value_tgo75
	use(value)
	return 7, nil
}
func contextassignWrapped(err *PointerError) (int, error) {
	var zero_tgo80 int
	defer done()
	value := 0
	var value_tgo78 int
	var error_tgo79 *PointerError
	value_tgo78, error_tgo79 = next(err)
	if error_tgo79 != nil {
		return zero_tgo80, fmt.Errorf("next: %w", error_tgo79)
	}
	value = value_tgo78
	use(value)
	return 7, nil
}
func contextreturnDirect(err *PointerError) (int, error) {
	var zero_tgo83 int
	defer done()
	var value_tgo81 int
	var error_tgo82 *PointerError
	value_tgo81, error_tgo82 = next(err)
	if error_tgo82 != nil {
		return zero_tgo83, error_tgo82
	}
	return use(value_tgo81), nil
}
func contextreturnWrapped(err *PointerError) (int, error) {
	var zero_tgo86 int
	defer done()
	var value_tgo84 int
	var error_tgo85 *PointerError
	value_tgo84, error_tgo85 = next(err)
	if error_tgo85 != nil {
		return zero_tgo86, fmt.Errorf("next: %w", error_tgo85)
	}
	return use(value_tgo84), nil
}
func contextloopInitDirect(err *PointerError) (int, error) {
	var zero_tgo89 int
	defer done()
	{
		var value_tgo87 int
		var error_tgo88 *PointerError
		value_tgo87, error_tgo88 = next(err)
		if error_tgo88 != nil {
			return zero_tgo89, error_tgo88
		}
		var tgo90 bool = 0 == 0
		for value := value_tgo87; ; {
			if tgo90 {
				tgo90 = 0 != 0
			} else {
				value++
			}
			if !(value < 8) {
				break
			}
			{
				use(value)
			}
		}
	}
	return 7, nil
}
func contextloopInitWrapped(err *PointerError) (int, error) {
	var zero_tgo93 int
	defer done()
	{
		var value_tgo91 int
		var error_tgo92 *PointerError
		value_tgo91, error_tgo92 = next(err)
		if error_tgo92 != nil {
			return zero_tgo93, fmt.Errorf("next: %w", error_tgo92)
		}
		var tgo94 bool = 0 == 0
		for value := value_tgo91; ; {
			if tgo94 {
				tgo94 = 0 != 0
			} else {
				value++
			}
			if !(value < 8) {
				break
			}
			{
				use(value)
			}
		}
	}
	return 7, nil
}
func contextloopPostDirect(err *PointerError) (int, error) {
	var zero_tgo97 int
	defer done()
	{
		var tgo98 bool = 0 == 0
		for value := 0; ; {
			if tgo98 {
				tgo98 = 0 != 0
			} else {
				var value_tgo95 int
				var error_tgo96 *PointerError
				value_tgo95, error_tgo96 = next(err)
				if error_tgo96 != nil {
					return zero_tgo97, error_tgo96
				}
				value = value_tgo95
			}
			if !(value < 1) {
				break
			}
			{
				use(value)
			}
		}
	}
	return 7, nil
}
func contextloopPostWrapped(err *PointerError) (int, error) {
	var zero_tgo101 int
	defer done()
	{
		var tgo102 bool = 0 == 0
		for value := 0; ; {
			if tgo102 {
				tgo102 = 0 != 0
			} else {
				var value_tgo99 int
				var error_tgo100 *PointerError
				value_tgo99, error_tgo100 = next(err)
				if error_tgo100 != nil {
					return zero_tgo101, fmt.Errorf("next: %w", error_tgo100)
				}
				value = value_tgo99
			}
			if !(value < 1) {
				break
			}
			{
				use(value)
			}
		}
	}
	return 7, nil
}
func contextswitchDirect(err *PointerError) (int, error) {
	var zero_tgo105 int
	defer done()
	{
		var value_tgo103 int
		var error_tgo104 *PointerError
		value_tgo103, error_tgo104 = next(err)
		if error_tgo104 != nil {
			return zero_tgo105, error_tgo104
		}
		switch value_tgo103 {
		case 7:
			use(7)
		}
	}
	return 7, nil
}
func contextswitchWrapped(err *PointerError) (int, error) {
	var zero_tgo108 int
	defer done()
	{
		var value_tgo106 int
		var error_tgo107 *PointerError
		value_tgo106, error_tgo107 = next(err)
		if error_tgo107 != nil {
			return zero_tgo108, fmt.Errorf("next: %w", error_tgo107)
		}
		switch value_tgo106 {
		case 7:
			use(7)
		}
	}
	return 7, nil
}
func contextcaseExprDirect(err *PointerError) (int, error) {
	var zero_tgo112 int
	defer done()
	{
		var tgo109 int = -1
		if tgo109 == -1 {
			var value_tgo110 int
			var error_tgo111 *PointerError
			value_tgo110, error_tgo111 = next(err)
			if error_tgo111 != nil {
				return zero_tgo112, error_tgo111
			}
			if 7 == value_tgo110 {
				tgo109 = 0
			}
		}
		switch tgo109 {
		case 0:
			use(7)
		}
	}
	return 7, nil
}
func contextcaseExprWrapped(err *PointerError) (int, error) {
	var zero_tgo116 int
	defer done()
	{
		var tgo113 int = -1
		if tgo113 == -1 {
			var value_tgo114 int
			var error_tgo115 *PointerError
			value_tgo114, error_tgo115 = next(err)
			if error_tgo115 != nil {
				return zero_tgo116, fmt.Errorf("next: %w", error_tgo115)
			}
			if 7 == value_tgo114 {
				tgo113 = 0
			}
		}
		switch tgo113 {
		case 0:
			use(7)
		}
	}
	return 7, nil
}
func contextshortCircuitDirect(err *PointerError) (int, error) {
	var zero_tgo119 int
	defer done()
	{
		var tgo120 bool
		tgo120 = false
		if tgo120 {
			var value_tgo117 int
			var error_tgo118 *PointerError
			value_tgo117, error_tgo118 = next(err)
			if error_tgo118 != nil {
				return zero_tgo119, error_tgo118
			}
			tgo120 = value_tgo117 == 7
		}
		if tgo120 == (0 == 0) {
			panic("skipped")
		}
	}
	use(7)
	return 7, nil
}
func contextshortCircuitWrapped(err *PointerError) (int, error) {
	var zero_tgo123 int
	defer done()
	{
		var tgo124 bool
		tgo124 = false
		if tgo124 {
			var value_tgo121 int
			var error_tgo122 *PointerError
			value_tgo121, error_tgo122 = next(err)
			if error_tgo122 != nil {
				return zero_tgo123, fmt.Errorf("next: %w", error_tgo122)
			}
			tgo124 = value_tgo121 == 7
		}
		if tgo124 == (0 == 0) {
			panic("skipped")
		}
	}
	use(7)
	return 7, nil
}
func contextliteralDirect(err *PointerError) (int, error) {
	defer done()
	run := func() error {
		var value_tgo125 int
		var error_tgo126 *PointerError
		value_tgo125, error_tgo126 = next(err)
		if error_tgo126 != nil {
			return error_tgo126
		}
		use(value_tgo125)
		return nil
	}
	return 0, run()
}
func contextliteralWrapped(err *PointerError) (int, error) {
	defer done()
	run := func() error {
		var value_tgo127 int
		var error_tgo128 *PointerError
		value_tgo127, error_tgo128 = next(err)
		if error_tgo128 != nil {
			return fmt.Errorf("next: %w", error_tgo128)
		}
		use(value_tgo127)
		return nil
	}
	return 0, run()
}
func constrainedPointerDirect[E interface {
	error
	*PointerError
}](err E) (int, error) {
	var zero_tgo131 int
	defer done()
	var value_tgo129 int
	var error_tgo130 E
	value_tgo129, error_tgo130 = next(err)
	if error_tgo130 != nil {
		return zero_tgo131, error_tgo130
	}
	value := use(value_tgo129)
	return value, nil
}
func constrainedPointerWrapped[E interface {
	error
	*PointerError
}](err E) (int, error) {
	var zero_tgo134 int
	defer done()
	var value_tgo132 int
	var error_tgo133 E
	value_tgo132, error_tgo133 = next(err)
	if error_tgo133 != nil {
		return zero_tgo134, fmt.Errorf("next: %w", error_tgo133)
	}
	value := use(value_tgo132)
	return value, nil
}
func constrainedNilableDirect[E interface {
	error
	~[]int | ~map[int]int
}](err E) (int, error) {
	var zero_tgo137 int
	defer done()
	var value_tgo135 int
	var error_tgo136 E
	value_tgo135, error_tgo136 = next(err)
	if error_tgo136 != nil {
		return zero_tgo137, error_tgo136
	}
	value := use(value_tgo135)
	return value, nil
}
func constrainedNilableWrapped[E interface {
	error
	~[]int | ~map[int]int
}](err E) (int, error) {
	var zero_tgo140 int
	defer done()
	var value_tgo138 int
	var error_tgo139 E
	value_tgo138, error_tgo139 = next(err)
	if error_tgo139 != nil {
		return zero_tgo140, fmt.Errorf("next: %w", error_tgo139)
	}
	value := use(value_tgo138)
	return value, nil
}
func constrainedMixedDirect[E interface {
	error
	*PointerError | NumberError
}](err E) (int, error) {
	var zero_tgo143 int
	defer done()
	var value_tgo141 int
	var error_tgo142 E
	value_tgo141, error_tgo142 = next(err)
	var reflected_tgo144 reflect_tgo52.Value = reflect_tgo52.ValueOf(&error_tgo142).Elem()
	var failed_tgo145 bool = 0 == 0
	switch reflected_tgo144.Kind() {
	case reflect_tgo52.Chan, reflect_tgo52.Func, reflect_tgo52.Interface, reflect_tgo52.Map, reflect_tgo52.Pointer, reflect_tgo52.Slice:
		failed_tgo145 = !reflected_tgo144.IsNil()
	}
	if failed_tgo145 {
		return zero_tgo143, error_tgo142
	}
	value := use(value_tgo141)
	return value, nil
}
func constrainedMixedWrapped[E interface {
	error
	*PointerError | NumberError
}](err E) (int, error) {
	var zero_tgo148 int
	defer done()
	var value_tgo146 int
	var error_tgo147 E
	value_tgo146, error_tgo147 = next(err)
	var reflected_tgo149 reflect_tgo52.Value = reflect_tgo52.ValueOf(&error_tgo147).Elem()
	var failed_tgo150 bool = 0 == 0
	switch reflected_tgo149.Kind() {
	case reflect_tgo52.Chan, reflect_tgo52.Func, reflect_tgo52.Interface, reflect_tgo52.Map, reflect_tgo52.Pointer, reflect_tgo52.Slice:
		failed_tgo150 = !reflected_tgo149.IsNil()
	}
	if failed_tgo150 {
		return zero_tgo148, fmt.Errorf("next: %w", error_tgo147)
	}
	value := use(value_tgo146)
	return value, nil
}
func shadowPointer(nil int, err *PointerError) (int, error) {
	var zero_tgo153 int
	defer done()
	var value_tgo151 int
	var error_tgo152 *PointerError
	value_tgo151, error_tgo152 = next(err)
	var zero_tgo154 *PointerError
	if error_tgo152 != zero_tgo154 {
		return zero_tgo153, error_tgo152
	}
	value := use(value_tgo151)
	var success error
	return value + nil, success
}
func main() { fmt.Print("ok") }
