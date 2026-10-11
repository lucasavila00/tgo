package main

import reflect_tgo53 "reflect"
import fmt_tgo7 "fmt"

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
	var tgo3 int
	defer done()
	var tgo1 int
	var tgo2 *PointerError
	tgo1, tgo2 = next(err)
	if tgo2 != nil {
		return tgo3, tgo2
	}
	value := use(tgo1)
	return value, nil
}

func pointerWrapped(err *PointerError) (int, error) {
	var tgo6 int
	defer done()
	var tgo4 int
	var tgo5 *PointerError
	tgo4, tgo5 = next(err)
	if tgo5 != nil {
		return tgo6, fmt_tgo7.Errorf("next: %w", tgo5)
	}
	value := use(tgo4)
	return value, nil
}

func ifaceDirect(err error) (int, error) {
	var tgo10 int
	defer done()
	var tgo8 int
	var tgo9 error
	tgo8, tgo9 = next(err)
	if tgo9 != nil {
		return tgo10, tgo9
	}
	value := use(tgo8)
	return value, nil
}

func ifaceWrapped(err error) (int, error) {
	var tgo13 int
	defer done()
	var tgo11 int
	var tgo12 error
	tgo11, tgo12 = next(err)
	if tgo12 != nil {
		return tgo13, fmt_tgo7.Errorf("next: %w", tgo12)
	}
	value := use(tgo11)
	return value, nil
}

func numberDirect(err NumberError) (int, error) {
	var tgo16 int
	defer done()
	var tgo14 int
	var tgo15 NumberError
	tgo14, tgo15 = next(err)
	if 0 == 0 {
		return tgo16, tgo15
	}
	value := use(tgo14)
	return value, nil
}

func numberWrapped(err NumberError) (int, error) {
	var tgo19 int
	defer done()
	var tgo17 int
	var tgo18 NumberError
	tgo17, tgo18 = next(err)
	if 0 == 0 {
		return tgo19, fmt_tgo7.Errorf("next: %w", tgo18)
	}
	value := use(tgo17)
	return value, nil
}

func sliceDirect(err SliceError) (int, error) {
	var tgo22 int
	defer done()
	var tgo20 int
	var tgo21 SliceError
	tgo20, tgo21 = next(err)
	if tgo21 != nil {
		return tgo22, tgo21
	}
	value := use(tgo20)
	return value, nil
}

func sliceWrapped(err SliceError) (int, error) {
	var tgo25 int
	defer done()
	var tgo23 int
	var tgo24 SliceError
	tgo23, tgo24 = next(err)
	if tgo24 != nil {
		return tgo25, fmt_tgo7.Errorf("next: %w", tgo24)
	}
	value := use(tgo23)
	return value, nil
}

func mappedDirect(err MapError) (int, error) {
	var tgo28 int
	defer done()
	var tgo26 int
	var tgo27 MapError
	tgo26, tgo27 = next(err)
	if tgo27 != nil {
		return tgo28, tgo27
	}
	value := use(tgo26)
	return value, nil
}

func mappedWrapped(err MapError) (int, error) {
	var tgo31 int
	defer done()
	var tgo29 int
	var tgo30 MapError
	tgo29, tgo30 = next(err)
	if tgo30 != nil {
		return tgo31, fmt_tgo7.Errorf("next: %w", tgo30)
	}
	value := use(tgo29)
	return value, nil
}

func channelDirect(err ChanError) (int, error) {
	var tgo34 int
	defer done()
	var tgo32 int
	var tgo33 ChanError
	tgo32, tgo33 = next(err)
	if tgo33 != nil {
		return tgo34, tgo33
	}
	value := use(tgo32)
	return value, nil
}

func channelWrapped(err ChanError) (int, error) {
	var tgo37 int
	defer done()
	var tgo35 int
	var tgo36 ChanError
	tgo35, tgo36 = next(err)
	if tgo36 != nil {
		return tgo37, fmt_tgo7.Errorf("next: %w", tgo36)
	}
	value := use(tgo35)
	return value, nil
}

func functionDirect(err FuncError) (int, error) {
	var tgo40 int
	defer done()
	var tgo38 int
	var tgo39 FuncError
	tgo38, tgo39 = next(err)
	if tgo39 != nil {
		return tgo40, tgo39
	}
	value := use(tgo38)
	return value, nil
}

func functionWrapped(err FuncError) (int, error) {
	var tgo43 int
	defer done()
	var tgo41 int
	var tgo42 FuncError
	tgo41, tgo42 = next(err)
	if tgo42 != nil {
		return tgo43, fmt_tgo7.Errorf("next: %w", tgo42)
	}
	value := use(tgo41)
	return value, nil
}

func structureDirect(err StructError) (int, error) {
	var tgo46 int
	defer done()
	var tgo44 int
	var tgo45 StructError
	tgo44, tgo45 = next(err)
	if 0 == 0 {
		return tgo46, tgo45
	}
	value := use(tgo44)
	return value, nil
}

func structureWrapped(err StructError) (int, error) {
	var tgo49 int
	defer done()
	var tgo47 int
	var tgo48 StructError
	tgo47, tgo48 = next(err)
	if 0 == 0 {
		return tgo49, fmt_tgo7.Errorf("next: %w", tgo48)
	}
	value := use(tgo47)
	return value, nil
}

func genericDirect[E error](err E) (int, error) {
	var tgo52 int
	defer done()
	var tgo50 int
	var tgo51 E
	tgo50, tgo51 = next(err)
	var tgo54 reflect_tgo53.Value = reflect_tgo53.ValueOf(&tgo51).Elem()
	var tgo55 bool = 0 == 0
	switch tgo54.Kind() {
	case reflect_tgo53.Chan, reflect_tgo53.Func, reflect_tgo53.Interface, reflect_tgo53.Map, reflect_tgo53.Pointer, reflect_tgo53.Slice:
		tgo55 = !tgo54.IsNil()
	}
	if tgo55 {
		return tgo52, tgo51
	}
	value := use(tgo50)
	return value, nil
}

func genericWrapped[E error](err E) (int, error) {
	var tgo58 int
	defer done()
	var tgo56 int
	var tgo57 E
	tgo56, tgo57 = next(err)
	var tgo59 reflect_tgo53.Value = reflect_tgo53.ValueOf(&tgo57).Elem()
	var tgo60 bool = 0 == 0
	switch tgo59.Kind() {
	case reflect_tgo53.Chan, reflect_tgo53.Func, reflect_tgo53.Interface, reflect_tgo53.Map, reflect_tgo53.Pointer, reflect_tgo53.Slice:
		tgo60 = !tgo59.IsNil()
	}
	if tgo60 {
		return tgo58, fmt_tgo7.Errorf("next: %w", tgo57)
	}
	value := use(tgo56)
	return value, nil
}

func comparableResult[E interface {
	error
	comparable
}](err E) (int, E) {
	var tgo63 int
	defer done()
	var tgo61 int
	var tgo62 E
	tgo61, tgo62 = next(err)
	var tgo64 reflect_tgo53.Value = reflect_tgo53.ValueOf(&tgo62).Elem()
	var tgo65 bool = 0 == 0
	switch tgo64.Kind() {
	case reflect_tgo53.Chan, reflect_tgo53.Func, reflect_tgo53.Interface, reflect_tgo53.Map, reflect_tgo53.Pointer, reflect_tgo53.Slice:
		tgo65 = !tgo64.IsNil()
	}
	if tgo65 {
		return tgo63, tgo62
	}
	value := use(tgo61)
	var success E
	return value, success
}
func shadow(nil int, err SliceError) (int, error) {
	var tgo68 int
	defer done()
	var tgo66 int
	var tgo67 SliceError
	tgo66, tgo67 = next(err)
	var tgo69 reflect_tgo53.Value = reflect_tgo53.ValueOf(&tgo67).Elem()
	var tgo70 bool = 0 == 0
	switch tgo69.Kind() {
	case reflect_tgo53.Chan, reflect_tgo53.Func, reflect_tgo53.Interface, reflect_tgo53.Map, reflect_tgo53.Pointer, reflect_tgo53.Slice:
		tgo70 = !tgo69.IsNil()
	}
	if tgo70 {
		return tgo68, tgo67
	}
	value := use(tgo66)
	var success error
	return value + nil, success
}
func genericReturn[E error](err E) (int, E) {
	var tgo73 int
	defer done()
	var tgo71 int
	var tgo72 E
	tgo71, tgo72 = next(err)
	var tgo74 reflect_tgo53.Value = reflect_tgo53.ValueOf(&tgo72).Elem()
	var tgo75 bool = 0 == 0
	switch tgo74.Kind() {
	case reflect_tgo53.Chan, reflect_tgo53.Func, reflect_tgo53.Interface, reflect_tgo53.Map, reflect_tgo53.Pointer, reflect_tgo53.Slice:
		tgo75 = !tgo74.IsNil()
	}
	if tgo75 {
		return tgo73, tgo72
	}
	value := use(tgo71)
	var success E
	return value, success
}
func contextassignDirect(err *PointerError) (int, error) {
	var tgo78 int
	defer done()
	value := 0
	{
		var tgo76 int
		var tgo77 *PointerError
		tgo76, tgo77 = next(err)
		if tgo77 != nil {
			return tgo78, tgo77
		}
		value = tgo76
	}
	use(value)
	return 7, nil
}
func contextassignWrapped(err *PointerError) (int, error) {
	var tgo81 int
	defer done()
	value := 0
	{
		var tgo79 int
		var tgo80 *PointerError
		tgo79, tgo80 = next(err)
		if tgo80 != nil {
			return tgo81, fmt_tgo7.Errorf("next: %w", tgo80)
		}
		value = tgo79
	}
	use(value)
	return 7, nil
}
func contextreturnDirect(err *PointerError) (int, error) {
	var tgo84 int
	defer done()
	{
		var tgo82 int
		var tgo83 *PointerError
		tgo82, tgo83 = next(err)
		if tgo83 != nil {
			return tgo84, tgo83
		}
		return use(tgo82), nil
	}
}
func contextreturnWrapped(err *PointerError) (int, error) {
	var tgo87 int
	defer done()
	{
		var tgo85 int
		var tgo86 *PointerError
		tgo85, tgo86 = next(err)
		if tgo86 != nil {
			return tgo87, fmt_tgo7.Errorf("next: %w", tgo86)
		}
		return use(tgo85), nil
	}
}
func contextloopInitDirect(err *PointerError) (int, error) {
	var tgo90 int
	defer done()
	{
		var tgo88 int
		var tgo89 *PointerError
		tgo88, tgo89 = next(err)
		if tgo89 != nil {
			return tgo90, tgo89
		}
		var tgo91 bool = 0 == 0
		for value := tgo88; ; {
			if tgo91 {
				tgo91 = 0 != 0
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
	var tgo94 int
	defer done()
	{
		var tgo92 int
		var tgo93 *PointerError
		tgo92, tgo93 = next(err)
		if tgo93 != nil {
			return tgo94, fmt_tgo7.Errorf("next: %w", tgo93)
		}
		var tgo95 bool = 0 == 0
		for value := tgo92; ; {
			if tgo95 {
				tgo95 = 0 != 0
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
	var tgo98 int
	defer done()
	{
		var tgo99 bool = 0 == 0
		for value := 0; ; {
			if tgo99 {
				tgo99 = 0 != 0
			} else {
				var tgo96 int
				var tgo97 *PointerError
				tgo96, tgo97 = next(err)
				if tgo97 != nil {
					return tgo98, tgo97
				}
				value = tgo96
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
	var tgo102 int
	defer done()
	{
		var tgo103 bool = 0 == 0
		for value := 0; ; {
			if tgo103 {
				tgo103 = 0 != 0
			} else {
				var tgo100 int
				var tgo101 *PointerError
				tgo100, tgo101 = next(err)
				if tgo101 != nil {
					return tgo102, fmt_tgo7.Errorf("next: %w", tgo101)
				}
				value = tgo100
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
	var tgo106 int
	defer done()
	{
		var tgo104 int
		var tgo105 *PointerError
		tgo104, tgo105 = next(err)
		if tgo105 != nil {
			return tgo106, tgo105
		}
		switch tgo104 {
		case 7:
			use(7)
		}
	}
	return 7, nil
}
func contextswitchWrapped(err *PointerError) (int, error) {
	var tgo109 int
	defer done()
	{
		var tgo107 int
		var tgo108 *PointerError
		tgo107, tgo108 = next(err)
		if tgo108 != nil {
			return tgo109, fmt_tgo7.Errorf("next: %w", tgo108)
		}
		switch tgo107 {
		case 7:
			use(7)
		}
	}
	return 7, nil
}
func contextcaseExprDirect(err *PointerError) (int, error) {
	var tgo113 int
	defer done()
	{
		var tgo110 int = -1
		if tgo110 == -1 {
			var tgo111 int
			var tgo112 *PointerError
			tgo111, tgo112 = next(err)
			if tgo112 != nil {
				return tgo113, tgo112
			}
			if 7 == tgo111 {
				tgo110 = 0
			}
		}
		switch tgo110 {
		case 0:
			use(7)
		}
	}
	return 7, nil
}
func contextcaseExprWrapped(err *PointerError) (int, error) {
	var tgo117 int
	defer done()
	{
		var tgo114 int = -1
		if tgo114 == -1 {
			var tgo115 int
			var tgo116 *PointerError
			tgo115, tgo116 = next(err)
			if tgo116 != nil {
				return tgo117, fmt_tgo7.Errorf("next: %w", tgo116)
			}
			if 7 == tgo115 {
				tgo114 = 0
			}
		}
		switch tgo114 {
		case 0:
			use(7)
		}
	}
	return 7, nil
}
func contextshortCircuitDirect(err *PointerError) (int, error) {
	var tgo120 int
	defer done()
	{
		var tgo121 bool
		tgo121 = false
		if tgo121 {
			var tgo118 int
			var tgo119 *PointerError
			tgo118, tgo119 = next(err)
			if tgo119 != nil {
				return tgo120, tgo119
			}
			tgo121 = tgo118 == 7
		}
		if tgo121 == (0 == 0) {
			panic("skipped")
		}
	}
	use(7)
	return 7, nil
}
func contextshortCircuitWrapped(err *PointerError) (int, error) {
	var tgo124 int
	defer done()
	{
		var tgo125 bool
		tgo125 = false
		if tgo125 {
			var tgo122 int
			var tgo123 *PointerError
			tgo122, tgo123 = next(err)
			if tgo123 != nil {
				return tgo124, fmt_tgo7.Errorf("next: %w", tgo123)
			}
			tgo125 = tgo122 == 7
		}
		if tgo125 == (0 == 0) {
			panic("skipped")
		}
	}
	use(7)
	return 7, nil
}
func contextliteralDirect(err *PointerError) (int, error) {
	defer done()
	run := func() error {
		{
			var tgo126 int
			var tgo127 *PointerError
			tgo126, tgo127 = next(err)
			if tgo127 != nil {
				return tgo127
			}
			use(tgo126)
		}
		return nil
	}
	return 0, run()
}
func contextliteralWrapped(err *PointerError) (int, error) {
	defer done()
	run := func() error {
		{
			var tgo128 int
			var tgo129 *PointerError
			tgo128, tgo129 = next(err)
			if tgo129 != nil {
				return fmt_tgo7.Errorf("next: %w", tgo129)
			}
			use(tgo128)
		}
		return nil
	}
	return 0, run()
}
func constrainedPointerDirect[E interface {
	error
	*PointerError
}](err E) (int, error) {
	var tgo132 int
	defer done()
	var tgo130 int
	var tgo131 E
	tgo130, tgo131 = next(err)
	if tgo131 != nil {
		return tgo132, tgo131
	}
	value := use(tgo130)
	return value, nil
}
func constrainedPointerWrapped[E interface {
	error
	*PointerError
}](err E) (int, error) {
	var tgo135 int
	defer done()
	var tgo133 int
	var tgo134 E
	tgo133, tgo134 = next(err)
	if tgo134 != nil {
		return tgo135, fmt_tgo7.Errorf("next: %w", tgo134)
	}
	value := use(tgo133)
	return value, nil
}
func constrainedNilableDirect[E interface {
	error
	~[]int | ~map[int]int
}](err E) (int, error) {
	var tgo138 int
	defer done()
	var tgo136 int
	var tgo137 E
	tgo136, tgo137 = next(err)
	if tgo137 != nil {
		return tgo138, tgo137
	}
	value := use(tgo136)
	return value, nil
}
func constrainedNilableWrapped[E interface {
	error
	~[]int | ~map[int]int
}](err E) (int, error) {
	var tgo141 int
	defer done()
	var tgo139 int
	var tgo140 E
	tgo139, tgo140 = next(err)
	if tgo140 != nil {
		return tgo141, fmt_tgo7.Errorf("next: %w", tgo140)
	}
	value := use(tgo139)
	return value, nil
}
func constrainedMixedDirect[E interface {
	error
	*PointerError | NumberError
}](err E) (int, error) {
	var tgo144 int
	defer done()
	var tgo142 int
	var tgo143 E
	tgo142, tgo143 = next(err)
	var tgo145 reflect_tgo53.Value = reflect_tgo53.ValueOf(&tgo143).Elem()
	var tgo146 bool = 0 == 0
	switch tgo145.Kind() {
	case reflect_tgo53.Chan, reflect_tgo53.Func, reflect_tgo53.Interface, reflect_tgo53.Map, reflect_tgo53.Pointer, reflect_tgo53.Slice:
		tgo146 = !tgo145.IsNil()
	}
	if tgo146 {
		return tgo144, tgo143
	}
	value := use(tgo142)
	return value, nil
}
func constrainedMixedWrapped[E interface {
	error
	*PointerError | NumberError
}](err E) (int, error) {
	var tgo149 int
	defer done()
	var tgo147 int
	var tgo148 E
	tgo147, tgo148 = next(err)
	var tgo150 reflect_tgo53.Value = reflect_tgo53.ValueOf(&tgo148).Elem()
	var tgo151 bool = 0 == 0
	switch tgo150.Kind() {
	case reflect_tgo53.Chan, reflect_tgo53.Func, reflect_tgo53.Interface, reflect_tgo53.Map, reflect_tgo53.Pointer, reflect_tgo53.Slice:
		tgo151 = !tgo150.IsNil()
	}
	if tgo151 {
		return tgo149, fmt_tgo7.Errorf("next: %w", tgo148)
	}
	value := use(tgo147)
	return value, nil
}
func shadowPointer(nil int, err *PointerError) (int, error) {
	var tgo154 int
	defer done()
	var tgo152 int
	var tgo153 *PointerError
	tgo152, tgo153 = next(err)
	var tgo155 *PointerError
	if tgo153 != tgo155 {
		return tgo154, tgo153
	}
	value := use(tgo152)
	var success error
	return value + nil, success
}
func main() { fmt.Print("ok") }
