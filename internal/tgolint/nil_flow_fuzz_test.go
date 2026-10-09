package tgolint

import (
	"fmt"
	"strings"
	"testing"
)

func FuzzNilAssignmentFlow(f *testing.F) {
	f.Add([]byte{0})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 64 {
			data = data[:64]
		}
		checkNilAssignmentProperty(t, data)
	})
}

func checkNilAssignmentProperty(t testing.TB, data []byte) {
	t.Helper()
	program := decodeNilProgram(data)
	cases := []struct {
		name   string
		body   string
		unsafe bool
	}{
		{
			name:   "direct",
			body:   program.directSource(),
			unsafe: program.unsafe(false),
		},
		{
			name:   "saved guard",
			body:   program.savedGuardSource(),
			unsafe: program.unsafe(true),
		},
	}
	for _, item := range cases {
		diagnostics, err := runNilAnalysis(
			t, "value, other, alias *Item, a, b bool", item.body,
		)
		if err != nil {
			t.Fatalf("%s: %v\n%s", item.name, err, item.body)
		}
		gotUnsafe := len(diagnostics) != 0
		if gotUnsafe != item.unsafe {
			t.Fatalf(
				"%s: got %d diagnostics; want unsafe=%t\n%s\n%s",
				item.name, len(diagnostics), item.unsafe, item.body,
				formatDiagnostics(diagnostics),
			)
		}
	}
}

type nilProgram struct {
	operations []byte
	mutation   byte
	target     byte
	condition  *flowCondition
}

func decodeNilProgram(data []byte) nilProgram {
	read := func(index int) byte {
		if index < len(data) {
			return data[index]
		}
		return 0
	}
	operationCount := int(read(0) % 8)
	operations := make([]byte, operationCount)
	for index := range operations {
		operations[index] = read(index + 1)
	}
	cursor := operationCount + 1
	conditionData := []byte(nil)
	if cursor+2 < len(data) {
		conditionData = data[cursor+2:]
	}
	return nilProgram{
		operations: operations,
		mutation:   read(cursor),
		target:     read(cursor+1) % 3,
		condition:  decodeFlowCondition(conditionData),
	}
}

func (p nilProgram) directSource() string {
	lines := p.operationSource(p.operations)
	lines = append(lines, fmt.Sprintf(
		"if %s { need(%s) }", p.condition.source(), flowVariable(p.target),
	))
	return strings.Join(lines, "\n")
}

func (p nilProgram) savedGuardSource() string {
	lines := p.operationSource(p.operations)
	lines = append(lines, "checked := "+p.condition.source())
	lines = append(lines, nilOperationSource(p.mutation))
	lines = append(lines, fmt.Sprintf(
		"if checked { need(%s) }", flowVariable(p.target),
	))
	return strings.Join(lines, "\n")
}

func (p nilProgram) operationSource(operations []byte) []string {
	lines := make([]string, len(operations))
	for index, operation := range operations {
		lines[index] = nilOperationSource(operation)
	}
	return lines
}

func nilOperationSource(operation byte) string {
	switch operation % 12 {
	case 0:
		return "value = nil"
	case 1:
		return "value = &Item{}"
	case 2:
		return "value = other"
	case 3:
		return "other = value"
	case 4:
		return "value, other = other, value"
	case 5:
		return "alias = value"
	case 6:
		return "value = alias"
	case 7:
		return "other = nil"
	case 8:
		return "other = &Item{}"
	case 9:
		return "alias = nil"
	case 10:
		return "alias = &Item{}"
	default:
		return "other, alias = alias, other"
	}
}

func (p nilProgram) unsafe(savedGuard bool) bool {
	for initial := byte(0); initial < 8; initial++ {
		state := [3]bool{initial&1 != 0, initial&2 != 0, initial&4 != 0}
		for _, operation := range p.operations {
			applyNilOperation(&state, operation)
		}
		conditionCanPass := p.condition.possible(true, state)
		if savedGuard {
			applyNilOperation(&state, p.mutation)
		}
		if conditionCanPass && !state[p.target] {
			return true
		}
	}
	return false
}

func applyNilOperation(state *[3]bool, operation byte) {
	switch operation % 12 {
	case 0:
		state[0] = false
	case 1:
		state[0] = true
	case 2:
		state[0] = state[1]
	case 3:
		state[1] = state[0]
	case 4:
		state[0], state[1] = state[1], state[0]
	case 5:
		state[2] = state[0]
	case 6:
		state[0] = state[2]
	case 7:
		state[1] = false
	case 8:
		state[1] = true
	case 9:
		state[2] = false
	case 10:
		state[2] = true
	default:
		state[1], state[2] = state[2], state[1]
	}
}

func flowVariable(index byte) string {
	return [...]string{"value", "other", "alias"}[index%3]
}

type flowCondition struct {
	kind        byte
	left, right *flowCondition
}

func decodeFlowCondition(data []byte) *flowCondition {
	index := 0
	var decode func(depth int) *flowCondition
	decode = func(depth int) *flowCondition {
		value := byte(0)
		if index < len(data) {
			value = data[index]
			index++
		}
		kind := value % 13
		if depth == 0 && kind >= 10 {
			kind %= 10
		}
		result := &flowCondition{kind: kind}
		switch kind {
		case 10, 11:
			result.left = decode(depth - 1)
			result.right = decode(depth - 1)
		case 12:
			result.left = decode(depth - 1)
		}
		return result
	}
	return decode(4)
}

func (c *flowCondition) source() string {
	if c.kind < 6 {
		operator := " == nil"
		if c.kind%2 == 1 {
			operator = " != nil"
		}
		return flowVariable(c.kind/2) + operator
	}
	switch c.kind {
	case 6:
		return "a"
	case 7:
		return "!a"
	case 8:
		return "b"
	case 9:
		return "!b"
	case 10:
		return "(" + c.left.source() + " && " + c.right.source() + ")"
	case 11:
		return "(" + c.left.source() + " || " + c.right.source() + ")"
	default:
		return "!(" + c.left.source() + ")"
	}
}

func (c *flowCondition) possible(result bool, state [3]bool) bool {
	if c.kind < 6 {
		nonNil := state[c.kind/2]
		actual := !nonNil
		if c.kind%2 == 1 {
			actual = nonNil
		}
		return result == actual
	}
	switch c.kind {
	case 6, 7, 8, 9:
		return true
	case 10:
		if result {
			return c.left.possible(true, state) && c.right.possible(true, state)
		}
		return c.left.possible(false, state) || c.right.possible(false, state)
	case 11:
		if result {
			return c.left.possible(true, state) || c.right.possible(true, state)
		}
		return c.left.possible(false, state) && c.right.possible(false, state)
	default:
		return c.left.possible(!result, state)
	}
}
