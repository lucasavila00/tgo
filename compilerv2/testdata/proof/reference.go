//go:build proofreference

package proof

// Reference states the expected effects without calling the fixture functions.
func Reference(name string, reached bool) []string {
	switch name {
	case "Calls":
		return []string{"left", "right", "consume:[1 1]", "pair", "consume:[1 2]"}
	case "Booleans":
		trace := []string{"left"}
		if reached {
			trace = append(trace, "right")
		}
		if reached {
			trace = append(trace, "observe:true")
		} else {
			trace = append(trace, "observe:false")
		}
		return append(trace, "named", "observe:true")
	case "Declarations":
		return []string{"first", "second", "shadow", "consume:[1 2]", "consume:[1 1]"}
	case "Assignments":
		return []string{"index", "key", "array value", "map value", "add", "consume:[2 1]"}
	case "AssignmentPanic":
		return []string{"bad index", "value before panic"}
	case "Loops":
		trace := []string{"init"}
		for i := 0; i < 2; i++ {
			trace = append(trace, "body", "post")
		}
		return append(trace, "consume:[0 1]")
	case "Switches":
		return []string{"tag", "case one", "matched", "fallthrough", "consume:[1]"}
	case "Ranges":
		return []string{"consume:[0]", "consume:[1]", "range target", "range body", "range target", "range body"}
	case "Selects":
		return []string{"send", "receive target", "received"}
	case "Scheduling":
		return []string{"defer argument", "go argument", "after wait", "consume:[1]"}
	case "Recover":
		return []string{"panic argument", "recovered"}
	case "Jumps":
		return []string{"labeled body", "labeled body", "labeled post", "labeled body", "labeled post"}
	case "Types":
		return []string{"receiver index", "method argument", "consume:[1 1]", "foreign argument", "foreign store", "consume:[1]"}
	case "Nested":
		return []string{"closure", "outside"}
	}
	panic("unknown reference case")
}

type markerRow struct {
	positions []int
	sites     [][2]int
}

// ReferenceMarkers records handwritten completion points, measured in effects.
func ReferenceMarkers(file string, start, end int, name string, reached bool) ([]int, bool) {
	var rows []markerRow
	var owner string
	switch file {
	case "calls.go":
		owner = "Calls"
		rows = []markerRow{
			{[]int{0}, [][2]int{{50, 57}, {58, 64}, {65, 71}}},
			{[]int{1}, [][2]int{{58, 72}, {74, 80}, {81, 88}}},
			{[]int{2}, [][2]int{{74, 89}}},
			{[]int{3}, [][2]int{{50, 90}, {92, 99}, {100, 104}}},
			{[]int{4}, [][2]int{{100, 106}}},
			{[]int{5}, [][2]int{{92, 107}, {116, 119}}},
		}
	case "booleans.go":
		owner = "Booleans"
		leftEnd := 1
		var rightBefore, rightAfter []int
		if reached {
			leftEnd = 2
			rightBefore = []int{1}
			rightAfter = []int{2}
		}
		rows = []markerRow{
			{[]int{0}, [][2]int{{69, 76}, {77, 90}, {91, 97}, {99, 106}}},
			{[]int{1}, [][2]int{{77, 107}}},
			{rightBefore, [][2]int{{111, 124}, {125, 132}, {134, 138}}},
			{rightAfter, [][2]int{{111, 139}}},
			{[]int{leftEnd}, [][2]int{{77, 139}}},
			{[]int{leftEnd + 1}, [][2]int{{69, 140}, {164, 177}, {178, 185}, {187, 191}}},
			{[]int{leftEnd + 2}, [][2]int{{159, 193}, {164, 192}, {195, 202}, {203, 214}, {208, 213}}},
			{[]int{leftEnd + 3}, [][2]int{{195, 215}, {224, 227}}},
		}
	case "declarations.go":
		owner = "Declarations"
		rows = []markerRow{
			{[]int{0}, [][2]int{{62, 68}, {69, 76}}},
			{[]int{1}, [][2]int{{62, 77}, {91, 97}, {98, 106}}},
			{[]int{2}, [][2]int{{91, 107}, {114, 119}, {114, 115}, {118, 119}, {133, 139}, {140, 148}}},
			{[]int{3}, [][2]int{{133, 149}, {152, 159}, {160, 161}, {163, 164}}},
			{[]int{4}, [][2]int{{152, 165}, {170, 177}, {178, 179}, {181, 182}}},
			{[]int{5}, [][2]int{{170, 183}, {192, 195}}},
		}
	case "assignments.go":
		if start < 250 {
			owner = "Assignments"
			rows = []markerRow{
				{[]int{0}, [][2]int{{61, 69}, {76, 78}, {77, 78}, {85, 98}, {100, 101}, {102, 108}, {109, 116}}},
				{[]int{1}, [][2]int{{100, 118}, {102, 117}, {120, 121}, {122, 128}, {129, 134}}},
				{[]int{2}, [][2]int{{120, 136}, {122, 135}, {139, 145}, {146, 159}}},
				{[]int{3}, [][2]int{{139, 160}, {162, 168}, {169, 180}}},
				{[]int{4}, [][2]int{{162, 181}, {183, 187}, {183, 184}, {185, 186}, {191, 197}, {198, 203}}},
				{[]int{5}, [][2]int{{191, 204}, {206, 213}, {214, 218}, {214, 215}, {216, 217}, {220, 224}, {220, 221}, {222, 223}}},
				{[]int{6}, [][2]int{{206, 225}, {234, 237}}},
			}
		} else {
			owner = "AssignmentPanic"
			rows = []markerRow{
				{[]int{0}, [][2]int{{291, 299}, {301, 302}, {303, 309}, {310, 321}}},
				{[]int{1}, [][2]int{{301, 323}, {303, 322}, {326, 332}, {333, 353}}},
				{[]int{2}, [][2]int{{326, 354}}},
				{nil, [][2]int{{363, 366}}},
			}
		}
	case "loops.go":
		owner = "Loops"
		rows = []markerRow{
			{[]int{0}, [][2]int{{86, 92}, {93, 99}}},
			{[]int{1}, [][2]int{{86, 104}, {86, 100}, {103, 104}}},
			{[]int{1, 3, 5}, [][2]int{{106, 111}, {106, 107}, {110, 111}}},
			{[]int{2, 4}, [][2]int{{113, 114}, {118, 124}, {125, 131}}},
			{[]int{3, 5}, [][2]int{{118, 132}}},
			{[]int{1, 3}, [][2]int{{137, 145}, {148, 189}, {155, 163}, {165, 188}, {192, 198}, {199, 205}}},
			{[]int{2, 4}, [][2]int{{192, 206}}},
			{[]int{5, 5}, [][2]int{{185, 186}}},
			{[]int{5}, [][2]int{{222, 229}, {230, 243}, {230, 241}, {230, 238}, {239, 240}, {245, 258}, {245, 256}, {245, 253}, {254, 255}}},
			{[]int{6}, [][2]int{{222, 259}, {268, 271}}},
		}
	case "jumps.go":
		owner = "Jumps"
		rows = []markerRow{
			{[]int{0}, [][2]int{{59, 64}}},
			{[]int{0, 1}, [][2]int{{81, 82}}},
			{[]int{0, 1, 3, 5}, [][2]int{{84, 89}, {84, 85}, {88, 89}}},
			{[]int{2, 4}, [][2]int{{91, 92}, {96, 102}, {103, 117}}},
			{[]int{3, 5}, [][2]int{{96, 118}}},
			{[]int{0, 1, 3}, [][2]int{{123, 129}, {130, 144}}},
			{[]int{1, 2, 4}, [][2]int{{123, 145}, {151, 157}, {152, 157}}},
			{[]int{1}, [][2]int{{163, 168}, {171, 175}}},
			{[]int{5}, [][2]int{{220, 223}}},
		}
	case "nested.go":
		owner = "Nested"
		rows = []markerRow{
			{[]int{0}, [][2]int{{59, 105}, {74, 80}, {81, 90}, {111, 115}}},
			{[]int{1}, [][2]int{{74, 91}, {100, 103}, {111, 117}, {119, 125}, {126, 135}}},
			{[]int{2}, [][2]int{{119, 136}, {145, 148}}},
		}
	case "ranges.go":
		owner = "Ranges"
		rows = []markerRow{
			{[]int{0, 1}, [][2]int{{85, 92}, {93, 94}}},
			{[]int{1, 2}, [][2]int{{85, 95}}},
			{[]int{2}, [][2]int{{155, 166}, {161, 162}, {164, 165}}},
			{[]int{2, 4}, [][2]int{{121, 122}, {123, 129}, {130, 144}}},
			{[]int{3, 5}, [][2]int{{121, 146}, {123, 145}, {171, 177}, {178, 190}}},
			{[]int{4, 6}, [][2]int{{171, 191}}},
			{[]int{6}, [][2]int{{203, 206}}},
		}
	case "switches.go":
		owner = "Switches"
		rows = []markerRow{
			{[]int{0}, [][2]int{{60, 66}, {67, 72}}},
			{[]int{1}, [][2]int{{60, 73}, {82, 88}, {89, 99}}},
			{[]int{2}, [][2]int{{82, 100}, {104, 110}, {111, 120}}},
			{[]int{3}, [][2]int{{104, 121}, {164, 170}, {171, 184}}},
			{nil, [][2]int{{142, 160}, {142, 148}, {149, 159}}},
			{[]int{4}, [][2]int{{164, 185}, {206, 207}, {225, 230}, {253, 260}, {261, 266}}},
			{[]int{5}, [][2]int{{253, 267}, {279, 282}}},
		}
	case "selects.go":
		owner = "Selects"
		rows = []markerRow{
			{[]int{0}, [][2]int{{61, 78}, {76, 77}, {80, 85}, {89, 95}, {96, 102}}},
			{[]int{1}, [][2]int{{89, 103}, {134, 135}, {136, 142}, {143, 159}, {164, 171}, {166, 171}}},
			{[]int{2}, [][2]int{{134, 161}, {136, 160}, {175, 181}, {182, 192}}},
			{[]int{3}, [][2]int{{175, 193}, {246, 249}}},
			{nil, [][2]int{{206, 234}, {206, 212}, {213, 233}}},
		}
	case "scheduling.go":
		if start < 350 {
			owner = "Scheduling"
			rows = []markerRow{
				{[]int{0}, [][2]int{{76, 83}, {84, 90}, {91, 107}}},
				{[]int{1}, [][2]int{{84, 108}, {119, 138}, {168, 182}, {168, 179}, {168, 175}, {180, 181}, {187, 240}, {241, 247}, {248, 261}}},
				{[]int{2}, [][2]int{{211, 223}, {211, 218}, {227, 238}, {233, 237}, {241, 262}, {265, 271}, {267, 271}, {273, 287}, {273, 285}, {273, 280}, {289, 295}, {296, 308}}},
				{[]int{3}, [][2]int{{289, 309}, {318, 321}}},
			}
		} else {
			owner = "Recover"
			rows = []markerRow{
				{[]int{0}, [][2]int{{368, 430}, {440, 446}, {447, 463}}},
				{[]int{1}, [][2]int{{382, 398}, {382, 391}, {395, 398}, {404, 410}, {411, 422}, {440, 464}}},
				{[]int{2}, [][2]int{{404, 423}}},
				{nil, [][2]int{{434, 465}}},
			}
		}
	case "types.go":
		owner = "Types"
		rows = []markerRow{
			{[]int{0}, [][2]int{{302, 313}, {302, 306}, {310, 313}, {320, 332}, {334, 335}, {336, 342}, {343, 359}}},
			{[]int{1}, [][2]int{{334, 367}, {334, 363}, {336, 362}, {336, 360}, {361, 362}, {368, 376}, {377, 383}, {384, 401}}},
			{[]int{2}, [][2]int{{156, 163}, {156, 157}, {167, 172}, {217, 222}, {334, 404}, {368, 403}, {377, 402}, {406, 413}, {414, 424}, {418, 423}, {426, 436}, {426, 430}, {426, 427}, {428, 429}}},
			{[]int{3}, [][2]int{{406, 437}, {439, 450}, {451, 464}, {465, 471}, {472, 490}}},
			{[]int{4}, [][2]int{{439, 493}, {451, 492}, {465, 491}, {495, 509}, {512, 518}, {519, 534}}},
			{[]int{5}, [][2]int{{512, 535}, {537, 544}, {545, 559}}},
			{[]int{6}, [][2]int{{537, 560}, {569, 572}}},
		}
	case "support.go":
		return supportMarkers(start, end, name, reached), true
	default:
		return nil, false
	}
	if name != owner {
		return nil, true
	}
	for _, row := range rows {
		for _, site := range row.sites {
			if site == [2]int{start, end} {
				return row.positions, true
			}
		}
	}
	panic("missing marker reference")
}

func supportMarkers(start, end int, name string, reached bool) []int {
	if start >= 600 {
		return []int{0}
	}
	helper := ""
	after := false
	switch {
	case start < 200:
		helper = "effect"
		after = start == 153
	case start < 300:
		helper = "boolean"
		after = start == 244
	case start < 390:
		helper = "pair"
		after = start == 341 || start == 344
	case start < 500:
		helper = "consume"
	default:
		helper = "observe"
	}
	var positions []int
	for i, event := range Reference(name, reached) {
		kind := "effect"
		if event == "pair" {
			kind = "pair"
		}
		if len(event) >= 8 && event[:8] == "consume:" {
			kind = "consume"
		}
		if len(event) >= 8 && event[:8] == "observe:" {
			kind = "observe"
		}
		if name == "Booleans" && (event == "left" || event == "right" || event == "named") {
			kind = "boolean"
		}
		if kind == helper {
			position := i
			if after {
				position++
			}
			positions = append(positions, position)
		}
	}
	return positions
}

// ReferenceReturn uses completion points and the source return boundaries.
func ReferenceReturn(file string, start, end int, name string, reached bool, visit int) ([]string, bool, bool) {
	trace := Reference(name, reached)
	positions, ok := ReferenceMarkers(file, start, end, name, reached)
	if !ok {
		panic("missing return reference")
	}
	if len(positions) < visit {
		return trace, false, name == "AssignmentPanic"
	}
	position := positions[visit-1]
	// The caller ignores the error from this closure and continues outside it.
	if file == "nested.go" && start >= 74 && end <= 103 {
		return append(trace[:position:position], "outside"), false, false
	}
	trace = append([]string(nil), trace[:position]...)
	// This deferred call is registered after its argument has been evaluated.
	if name == "Scheduling" && start >= 119 {
		trace = append(trace, "consume:[1]")
	}
	return trace, true, false
}
