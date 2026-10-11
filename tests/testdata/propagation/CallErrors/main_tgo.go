package main

import tgo104 "reflect"
import tgo13 "fmt"

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

func pointerDirect(err *PointerError) (tgo3 int, tgo5 error) {
	var tgo4 int = tgo3
	var tgo6 error = tgo5

	defer done()
	var tgo1 int
	var tgo2 *PointerError
	tgo1, tgo2 = next(err)
	if tgo2 != nil {
		tgo6 = tgo2
		return tgo4, tgo6
	}
	value := use(tgo1)
	return value, nil
}

func pointerWrapped(err *PointerError) (tgo9 int, tgo11 error) {
	var tgo10 int = tgo9
	var tgo12 error = tgo11

	defer done()
	var tgo7 int
	var tgo8 *PointerError
	tgo7, tgo8 = next(err)
	if tgo8 != nil {
		tgo12 = tgo8
		tgo12 = tgo13.Errorf("%s: %w", "next", tgo12)
		return tgo10, tgo12
	}
	value := use(tgo7)
	return value, nil
}

func ifaceDirect(err error) (tgo16 int, tgo18 error) {
	var tgo17 int = tgo16
	var tgo19 error = tgo18

	defer done()
	var tgo14 int
	var tgo15 error
	tgo14, tgo15 = next(err)
	if tgo15 != nil {
		tgo19 = tgo15
		return tgo17, tgo19
	}
	value := use(tgo14)
	return value, nil
}

func ifaceWrapped(err error) (tgo22 int, tgo24 error) {
	var tgo23 int = tgo22
	var tgo25 error = tgo24

	defer done()
	var tgo20 int
	var tgo21 error
	tgo20, tgo21 = next(err)
	if tgo21 != nil {
		tgo25 = tgo21
		tgo25 = tgo13.Errorf("%s: %w", "next", tgo25)
		return tgo23, tgo25
	}
	value := use(tgo20)
	return value, nil
}

func numberDirect(err NumberError) (tgo28 int, tgo30 error) {
	var tgo29 int = tgo28
	var tgo31 error = tgo30

	defer done()
	var tgo26 int
	var tgo27 NumberError
	tgo26, tgo27 = next(err)
	if 0 == 0 {
		tgo31 = tgo27
		return tgo29, tgo31
	}
	value := use(tgo26)
	return value, nil
}

func numberWrapped(err NumberError) (tgo34 int, tgo36 error) {
	var tgo35 int = tgo34
	var tgo37 error = tgo36

	defer done()
	var tgo32 int
	var tgo33 NumberError
	tgo32, tgo33 = next(err)
	if 0 == 0 {
		tgo37 = tgo33
		tgo37 = tgo13.Errorf("%s: %w", "next", tgo37)
		return tgo35, tgo37
	}
	value := use(tgo32)
	return value, nil
}

func sliceDirect(err SliceError) (tgo40 int, tgo42 error) {
	var tgo41 int = tgo40
	var tgo43 error = tgo42

	defer done()
	var tgo38 int
	var tgo39 SliceError
	tgo38, tgo39 = next(err)
	if tgo39 != nil {
		tgo43 = tgo39
		return tgo41, tgo43
	}
	value := use(tgo38)
	return value, nil
}

func sliceWrapped(err SliceError) (tgo46 int, tgo48 error) {
	var tgo47 int = tgo46
	var tgo49 error = tgo48

	defer done()
	var tgo44 int
	var tgo45 SliceError
	tgo44, tgo45 = next(err)
	if tgo45 != nil {
		tgo49 = tgo45
		tgo49 = tgo13.Errorf("%s: %w", "next", tgo49)
		return tgo47, tgo49
	}
	value := use(tgo44)
	return value, nil
}

func mappedDirect(err MapError) (tgo52 int, tgo54 error) {
	var tgo53 int = tgo52
	var tgo55 error = tgo54

	defer done()
	var tgo50 int
	var tgo51 MapError
	tgo50, tgo51 = next(err)
	if tgo51 != nil {
		tgo55 = tgo51
		return tgo53, tgo55
	}
	value := use(tgo50)
	return value, nil
}

func mappedWrapped(err MapError) (tgo58 int, tgo60 error) {
	var tgo59 int = tgo58
	var tgo61 error = tgo60

	defer done()
	var tgo56 int
	var tgo57 MapError
	tgo56, tgo57 = next(err)
	if tgo57 != nil {
		tgo61 = tgo57
		tgo61 = tgo13.Errorf("%s: %w", "next", tgo61)
		return tgo59, tgo61
	}
	value := use(tgo56)
	return value, nil
}

func channelDirect(err ChanError) (tgo64 int, tgo66 error) {
	var tgo65 int = tgo64
	var tgo67 error = tgo66

	defer done()
	var tgo62 int
	var tgo63 ChanError
	tgo62, tgo63 = next(err)
	if tgo63 != nil {
		tgo67 = tgo63
		return tgo65, tgo67
	}
	value := use(tgo62)
	return value, nil
}

func channelWrapped(err ChanError) (tgo70 int, tgo72 error) {
	var tgo71 int = tgo70
	var tgo73 error = tgo72

	defer done()
	var tgo68 int
	var tgo69 ChanError
	tgo68, tgo69 = next(err)
	if tgo69 != nil {
		tgo73 = tgo69
		tgo73 = tgo13.Errorf("%s: %w", "next", tgo73)
		return tgo71, tgo73
	}
	value := use(tgo68)
	return value, nil
}

func functionDirect(err FuncError) (tgo76 int, tgo78 error) {
	var tgo77 int = tgo76
	var tgo79 error = tgo78

	defer done()
	var tgo74 int
	var tgo75 FuncError
	tgo74, tgo75 = next(err)
	if tgo75 != nil {
		tgo79 = tgo75
		return tgo77, tgo79
	}
	value := use(tgo74)
	return value, nil
}

func functionWrapped(err FuncError) (tgo82 int, tgo84 error) {
	var tgo83 int = tgo82
	var tgo85 error = tgo84

	defer done()
	var tgo80 int
	var tgo81 FuncError
	tgo80, tgo81 = next(err)
	if tgo81 != nil {
		tgo85 = tgo81
		tgo85 = tgo13.Errorf("%s: %w", "next", tgo85)
		return tgo83, tgo85
	}
	value := use(tgo80)
	return value, nil
}

func structureDirect(err StructError) (tgo88 int, tgo90 error) {
	var tgo89 int = tgo88
	var tgo91 error = tgo90

	defer done()
	var tgo86 int
	var tgo87 StructError
	tgo86, tgo87 = next(err)
	if 0 == 0 {
		tgo91 = tgo87
		return tgo89, tgo91
	}
	value := use(tgo86)
	return value, nil
}

func structureWrapped(err StructError) (tgo94 int, tgo96 error) {
	var tgo95 int = tgo94
	var tgo97 error = tgo96

	defer done()
	var tgo92 int
	var tgo93 StructError
	tgo92, tgo93 = next(err)
	if 0 == 0 {
		tgo97 = tgo93
		tgo97 = tgo13.Errorf("%s: %w", "next", tgo97)
		return tgo95, tgo97
	}
	value := use(tgo92)
	return value, nil
}

func genericDirect[E error](err E) (tgo100 int, tgo102 error) {
	var tgo101 int = tgo100
	var tgo103 error = tgo102

	defer done()
	var tgo98 int
	var tgo99 E
	tgo98, tgo99 = next(err)
	var tgo105 tgo104.Value = tgo104.ValueOf(&tgo99).Elem()
	var tgo106 bool = 0 == 0
	switch tgo105.Kind() {
	case tgo104.Chan, tgo104.Func, tgo104.Interface, tgo104.Map, tgo104.Pointer, tgo104.Slice:
		tgo106 = !tgo105.IsNil()
	}
	if tgo106 {
		tgo103 = tgo99
		return tgo101, tgo103
	}

	value := use(tgo98)
	return value, nil
}

func genericWrapped[E error](err E) (tgo109 int, tgo111 error) {
	var tgo110 int = tgo109
	var tgo112 error = tgo111

	defer done()
	var tgo107 int
	var tgo108 E
	tgo107, tgo108 = next(err)
	var tgo113 tgo104.Value = tgo104.ValueOf(&tgo108).Elem()
	var tgo114 bool = 0 == 0
	switch tgo113.Kind() {
	case tgo104.Chan, tgo104.Func, tgo104.Interface, tgo104.Map, tgo104.Pointer, tgo104.Slice:
		tgo114 = !tgo113.IsNil()
	}
	if tgo114 {
		tgo112 = tgo108
		tgo112 = tgo13.Errorf("%s: %w", "next", tgo112)
		return tgo110, tgo112
	}

	value := use(tgo107)
	return value, nil
}

func comparableResult[E interface {
	error
	comparable
}](err E) (tgo117 int, tgo119 E) {
	var tgo118 int = tgo117
	var tgo120 E = tgo119

	defer done()
	var tgo115 int
	var tgo116 E
	tgo115, tgo116 = next(err)
	var tgo121 tgo104.Value = tgo104.ValueOf(&tgo116).Elem()
	var tgo122 bool = 0 == 0
	switch tgo121.Kind() {
	case tgo104.Chan, tgo104.Func, tgo104.Interface, tgo104.Map, tgo104.Pointer, tgo104.Slice:
		tgo122 = !tgo121.IsNil()
	}
	if tgo122 {
		tgo120 = tgo116
		return tgo118, tgo120
	}

	value := use(tgo115)
	var success E
	return value, success
}
func shadow(nil int, err SliceError) (tgo125 int, tgo127 error) {
	var tgo126 int = tgo125
	var tgo128 error = tgo127

	defer done()
	var tgo123 int
	var tgo124 SliceError
	tgo123, tgo124 = next(err)
	var tgo129 tgo104.Value = tgo104.ValueOf(&tgo124).Elem()
	var tgo130 bool = 0 == 0
	switch tgo129.Kind() {
	case tgo104.Chan, tgo104.Func, tgo104.Interface, tgo104.Map, tgo104.Pointer, tgo104.Slice:
		tgo130 = !tgo129.IsNil()
	}
	if tgo130 {
		tgo128 = tgo124
		return tgo126, tgo128
	}

	value := use(tgo123)
	var success error
	return value + nil, success
}
func genericReturn[E error](err E) (tgo133 int, tgo135 E) {
	var tgo134 int = tgo133
	var tgo136 E = tgo135

	defer done()
	var tgo131 int
	var tgo132 E
	tgo131, tgo132 = next(err)
	var tgo137 tgo104.Value = tgo104.ValueOf(&tgo132).Elem()
	var tgo138 bool = 0 == 0
	switch tgo137.Kind() {
	case tgo104.Chan, tgo104.Func, tgo104.Interface, tgo104.Map, tgo104.Pointer, tgo104.Slice:
		tgo138 = !tgo137.IsNil()
	}
	if tgo138 {
		tgo136 = tgo132
		return tgo134, tgo136
	}

	value := use(tgo131)
	var success E
	return value, success
}
func contextassignDirect(err *PointerError) (tgo141 int, tgo143 error) {
	var tgo142 int = tgo141
	var tgo144 error = tgo143

	defer done()
	value := 0
	{
		var tgo139 int
		var tgo140 *PointerError
		tgo139, tgo140 = next(err)
		if tgo140 != nil {
			tgo144 = tgo140
			return tgo142, tgo144
		}
		value = tgo139
	}
	use(value)
	return 7, nil
}
func contextassignWrapped(err *PointerError) (tgo147 int, tgo149 error) {
	var tgo148 int = tgo147
	var tgo150 error = tgo149

	defer done()
	value := 0
	{
		var tgo145 int
		var tgo146 *PointerError
		tgo145, tgo146 = next(err)
		if tgo146 != nil {
			tgo150 = tgo146
			tgo150 = tgo13.Errorf("%s: %w", "next", tgo150)
			return tgo148, tgo150
		}
		value = tgo145
	}
	use(value)
	return 7, nil
}
func contextreturnDirect(err *PointerError) (tgo153 int, tgo155 error) {
	var tgo154 int = tgo153
	var tgo156 error = tgo155

	defer done()
	{
		var tgo151 int
		var tgo152 *PointerError
		tgo151, tgo152 = next(err)
		if tgo152 != nil {
			tgo156 = tgo152
			return tgo154, tgo156
		}
		return use(tgo151), nil
	}
}
func contextreturnWrapped(err *PointerError) (tgo159 int, tgo161 error) {
	var tgo160 int = tgo159
	var tgo162 error = tgo161

	defer done()
	{
		var tgo157 int
		var tgo158 *PointerError
		tgo157, tgo158 = next(err)
		if tgo158 != nil {
			tgo162 = tgo158
			tgo162 = tgo13.Errorf("%s: %w", "next", tgo162)
			return tgo160, tgo162
		}
		return use(tgo157), nil
	}
}
func contextloopInitDirect(err *PointerError) (tgo165 int, tgo167 error) {
	var tgo166 int = tgo165
	var tgo168 error = tgo167

	defer done()
	{
		var tgo163 int
		var tgo164 *PointerError
		tgo163, tgo164 = next(err)
		if tgo164 != nil {
			tgo168 = tgo164
			return tgo166, tgo168
		}
		var tgo169 bool = 0 == 0

		for value := tgo163; ; {
			if tgo169 {
				tgo169 = 0 != 0
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
func contextloopInitWrapped(err *PointerError) (tgo172 int, tgo174 error) {
	var tgo173 int = tgo172
	var tgo175 error = tgo174

	defer done()
	{
		var tgo170 int
		var tgo171 *PointerError
		tgo170, tgo171 = next(err)
		if tgo171 != nil {
			tgo175 = tgo171
			tgo175 = tgo13.Errorf("%s: %w", "next", tgo175)
			return tgo173, tgo175
		}
		var tgo176 bool = 0 == 0

		for value := tgo170; ; {
			if tgo176 {
				tgo176 = 0 != 0
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
func contextloopPostDirect(err *PointerError) (tgo179 int, tgo181 error) {
	var tgo180 int = tgo179
	var tgo182 error = tgo181

	defer done()
	{
		var tgo183 bool = 0 == 0

		for value := 0; ; {
			if tgo183 {
				tgo183 = 0 != 0
			} else {
				var tgo177 int
				var tgo178 *PointerError
				tgo177, tgo178 = next(err)
				if tgo178 != nil {
					tgo182 = tgo178
					return tgo180, tgo182
				}
				value = tgo177
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
func contextloopPostWrapped(err *PointerError) (tgo186 int, tgo188 error) {
	var tgo187 int = tgo186
	var tgo189 error = tgo188

	defer done()
	{
		var tgo190 bool = 0 == 0

		for value := 0; ; {
			if tgo190 {
				tgo190 = 0 != 0
			} else {
				var tgo184 int
				var tgo185 *PointerError
				tgo184, tgo185 = next(err)
				if tgo185 != nil {
					tgo189 = tgo185
					tgo189 = tgo13.Errorf("%s: %w", "next", tgo189)
					return tgo187, tgo189
				}
				value = tgo184
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
func contextswitchDirect(err *PointerError) (tgo193 int, tgo195 error) {
	var tgo194 int = tgo193
	var tgo196 error = tgo195

	defer done()
	{
		var tgo191 int
		var tgo192 *PointerError
		tgo191, tgo192 = next(err)
		if tgo192 != nil {
			tgo196 = tgo192
			return tgo194, tgo196
		}
		switch tgo191 {
		case 7:
			use(7)
		}
	}

	return 7, nil
}
func contextswitchWrapped(err *PointerError) (tgo199 int, tgo201 error) {
	var tgo200 int = tgo199
	var tgo202 error = tgo201

	defer done()
	{
		var tgo197 int
		var tgo198 *PointerError
		tgo197, tgo198 = next(err)
		if tgo198 != nil {
			tgo202 = tgo198
			tgo202 = tgo13.Errorf("%s: %w", "next", tgo202)
			return tgo200, tgo202
		}
		switch tgo197 {
		case 7:
			use(7)
		}
	}

	return 7, nil
}
func contextcaseExprDirect(err *PointerError) (tgo206 int, tgo208 error) {
	var tgo207 int = tgo206
	var tgo209 error = tgo208

	defer done()
	{
		var tgo203 int = -1
		if tgo203 == -1 {
			var tgo204 int
			var tgo205 *PointerError
			tgo204, tgo205 = next(err)
			if tgo205 != nil {
				tgo209 = tgo205
				return tgo207, tgo209
			}
			if 7 == tgo204 {
				tgo203 = 0
			}
		}
		switch tgo203 {
		case 0:
			use(7)
		}
	}

	return 7, nil
}
func contextcaseExprWrapped(err *PointerError) (tgo213 int, tgo215 error) {
	var tgo214 int = tgo213
	var tgo216 error = tgo215

	defer done()
	{
		var tgo210 int = -1
		if tgo210 == -1 {
			var tgo211 int
			var tgo212 *PointerError
			tgo211, tgo212 = next(err)
			if tgo212 != nil {
				tgo216 = tgo212
				tgo216 = tgo13.Errorf("%s: %w", "next", tgo216)
				return tgo214, tgo216
			}
			if 7 == tgo211 {
				tgo210 = 0
			}
		}
		switch tgo210 {
		case 0:
			use(7)
		}
	}

	return 7, nil
}
func contextshortCircuitDirect(err *PointerError) (tgo219 int, tgo221 error) {
	var tgo220 int = tgo219
	var tgo222 error = tgo221

	defer done()
	{
		var tgo223 bool
		tgo223 = false
		if tgo223 {
			var tgo217 int
			var tgo218 *PointerError
			tgo217, tgo218 = next(err)
			if tgo218 != nil {
				tgo222 = tgo218
				return tgo220, tgo222
			}
			tgo223 = tgo217 == 7
		}
		if tgo223 == (0 == 0) {
			panic("skipped")
		}
	}
	use(7)
	return 7, nil
}
func contextshortCircuitWrapped(err *PointerError) (tgo226 int, tgo228 error) {
	var tgo227 int = tgo226
	var tgo229 error = tgo228

	defer done()
	{
		var tgo230 bool
		tgo230 = false
		if tgo230 {
			var tgo224 int
			var tgo225 *PointerError
			tgo224, tgo225 = next(err)
			if tgo225 != nil {
				tgo229 = tgo225
				tgo229 = tgo13.Errorf("%s: %w", "next", tgo229)
				return tgo227, tgo229
			}
			tgo230 = tgo224 == 7
		}
		if tgo230 == (0 == 0) {
			panic("skipped")
		}
	}
	use(7)
	return 7, nil
}
func contextliteralDirect(err *PointerError) (int, error) {
	defer done()
	run := func() (tgo233 error) {
		var tgo234 error = tgo233
		{
			var tgo231 int
			var tgo232 *PointerError
			tgo231, tgo232 = next(err)
			if tgo232 != nil {
				tgo234 = tgo232
				return tgo234
			}
			use(tgo231)
		}
		return nil
	}
	return 0, run()
}
func contextliteralWrapped(err *PointerError) (int, error) {
	defer done()
	run := func() (tgo237 error) {
		var tgo238 error = tgo237
		{
			var tgo235 int
			var tgo236 *PointerError
			tgo235, tgo236 = next(err)
			if tgo236 != nil {
				tgo238 = tgo236
				tgo238 = tgo13.Errorf("%s: %w", "next", tgo238)
				return tgo238
			}
			use(tgo235)
		}
		return nil
	}
	return 0, run()
}
func constrainedPointerDirect[E interface {
	error
	*PointerError
}](err E) (tgo241 int, tgo243 error) {
	var tgo242 int = tgo241
	var tgo244 error = tgo243

	defer done()
	var tgo239 int
	var tgo240 E
	tgo239, tgo240 = next(err)
	if tgo240 != nil {
		tgo244 = tgo240
		return tgo242, tgo244
	}
	value := use(tgo239)
	return value, nil
}
func constrainedPointerWrapped[E interface {
	error
	*PointerError
}](err E) (tgo247 int, tgo249 error) {
	var tgo248 int = tgo247
	var tgo250 error = tgo249

	defer done()
	var tgo245 int
	var tgo246 E
	tgo245, tgo246 = next(err)
	if tgo246 != nil {
		tgo250 = tgo246
		tgo250 = tgo13.Errorf("%s: %w", "next", tgo250)
		return tgo248, tgo250
	}
	value := use(tgo245)
	return value, nil
}
func constrainedNilableDirect[E interface {
	error
	~[]int | ~map[int]int
}](err E) (tgo253 int, tgo255 error) {
	var tgo254 int = tgo253
	var tgo256 error = tgo255

	defer done()
	var tgo251 int
	var tgo252 E
	tgo251, tgo252 = next(err)
	if tgo252 != nil {
		tgo256 = tgo252
		return tgo254, tgo256
	}
	value := use(tgo251)
	return value, nil
}
func constrainedNilableWrapped[E interface {
	error
	~[]int | ~map[int]int
}](err E) (tgo259 int, tgo261 error) {
	var tgo260 int = tgo259
	var tgo262 error = tgo261

	defer done()
	var tgo257 int
	var tgo258 E
	tgo257, tgo258 = next(err)
	if tgo258 != nil {
		tgo262 = tgo258
		tgo262 = tgo13.Errorf("%s: %w", "next", tgo262)
		return tgo260, tgo262
	}
	value := use(tgo257)
	return value, nil
}
func constrainedMixedDirect[E interface {
	error
	*PointerError | NumberError
}](err E) (tgo265 int, tgo267 error) {
	var tgo266 int = tgo265
	var tgo268 error = tgo267

	defer done()
	var tgo263 int
	var tgo264 E
	tgo263, tgo264 = next(err)
	var tgo269 tgo104.Value = tgo104.ValueOf(&tgo264).Elem()
	var tgo270 bool = 0 == 0
	switch tgo269.Kind() {
	case tgo104.Chan, tgo104.Func, tgo104.Interface, tgo104.Map, tgo104.Pointer, tgo104.Slice:
		tgo270 = !tgo269.IsNil()
	}
	if tgo270 {
		tgo268 = tgo264
		return tgo266, tgo268
	}

	value := use(tgo263)
	return value, nil
}
func constrainedMixedWrapped[E interface {
	error
	*PointerError | NumberError
}](err E) (tgo273 int, tgo275 error) {
	var tgo274 int = tgo273
	var tgo276 error = tgo275

	defer done()
	var tgo271 int
	var tgo272 E
	tgo271, tgo272 = next(err)
	var tgo277 tgo104.Value = tgo104.ValueOf(&tgo272).Elem()
	var tgo278 bool = 0 == 0
	switch tgo277.Kind() {
	case tgo104.Chan, tgo104.Func, tgo104.Interface, tgo104.Map, tgo104.Pointer, tgo104.Slice:
		tgo278 = !tgo277.IsNil()
	}
	if tgo278 {
		tgo276 = tgo272
		tgo276 = tgo13.Errorf("%s: %w", "next", tgo276)
		return tgo274, tgo276
	}

	value := use(tgo271)
	return value, nil
}
func shadowPointer(nil int, err *PointerError) (tgo281 int, tgo283 error) {
	var tgo282 int = tgo281
	var tgo284 error = tgo283

	defer done()
	var tgo279 int
	var tgo280 *PointerError
	tgo279, tgo280 = next(err)
	var tgo285 *PointerError
	if tgo280 != tgo285 {
		tgo284 = tgo280
		return tgo282, tgo284
	}

	value := use(tgo279)
	var success error
	return value + nil, success
}
func main() { fmt.Print("ok") }
