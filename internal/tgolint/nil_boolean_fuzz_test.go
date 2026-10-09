package tgolint

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestNilBooleanNarrowingProperties(t *testing.T) {
	random := rand.New(rand.NewSource(2)) //nolint:gosec // Tests need stable data.
	for range 300 {
		data := make([]byte, 12)
		if _, err := random.Read(data); err != nil {
			t.Fatal(err)
		}
		checkNilBooleanProperty(t, data)
	}
}

func FuzzNilBooleanNarrowing(f *testing.F) {
	f.Add([]byte{0})
	f.Add([]byte{1})
	f.Add([]byte{6, 1, 0})
	f.Add([]byte{7, 0, 1})
	f.Add([]byte{8, 6, 0, 1})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 64 {
			data = data[:64]
		}
		checkNilBooleanProperty(t, data)
	})
}

func checkNilBooleanProperty(t testing.TB, data []byte) {
	t.Helper()
	condition := decodeNilCondition(data)
	expression := condition.source()

	cases := []struct {
		name   string
		body   string
		unsafe bool
	}{
		{
			name:   "branch",
			body:   fmt.Sprintf("if %s { need(value) }", expression),
			unsafe: condition.possible(true, false),
		},
		{
			name: "guard alias",
			body: fmt.Sprintf(
				"guard := %s\nif guard { need(value) }", expression,
			),
			unsafe: condition.possible(true, false),
		},
		{
			name:   "early exit",
			body:   fmt.Sprintf("if %s { return }\nneed(value)", expression),
			unsafe: condition.possible(false, false),
		},
		{
			name:   "else branch",
			body:   fmt.Sprintf("if %s {} else { need(value) }", expression),
			unsafe: condition.possible(false, false),
		},
		{
			name: "loop body",
			body: fmt.Sprintf(
				"for %s { need(value); break }", expression,
			),
			unsafe: condition.possible(true, false),
		},
	}

	for _, item := range cases {
		diagnostics, err := runNilAnalysis(
			"value, other, alias *Item, a, b bool", item.body,
		)
		if err != nil {
			t.Fatalf("%s: %v\ncondition: %s", item.name, err, expression)
		}
		gotUnsafe := len(diagnostics) != 0
		if gotUnsafe != item.unsafe {
			t.Fatalf(
				"%s: got %d diagnostics; want unsafe=%t\ncondition: %s\n%s",
				item.name, len(diagnostics), item.unsafe, expression,
				formatDiagnostics(diagnostics),
			)
		}
	}
}

type nilCondition struct {
	kind        byte
	left, right *nilCondition
}

func decodeNilCondition(data []byte) *nilCondition {
	index := 0
	var decode func(depth int) *nilCondition
	decode = func(depth int) *nilCondition {
		value := byte(0)
		if index < len(data) {
			value = data[index]
			index++
		}
		kind := value % 9
		if depth == 0 && kind >= 6 {
			kind %= 6
		}
		result := &nilCondition{kind: kind}
		switch kind {
		case 6, 7:
			result.left = decode(depth - 1)
			result.right = decode(depth - 1)
		case 8:
			result.left = decode(depth - 1)
		}
		return result
	}
	return decode(4)
}

func (c *nilCondition) source() string {
	switch c.kind {
	case 0:
		return "value == nil"
	case 1:
		return "value != nil"
	case 2:
		return "a"
	case 3:
		return "!a"
	case 4:
		return "b"
	case 5:
		return "!b"
	case 6:
		return "(" + c.left.source() + " && " + c.right.source() + ")"
	case 7:
		return "(" + c.left.source() + " || " + c.right.source() + ")"
	default:
		return "!(" + c.left.source() + ")"
	}
}

// possible reports whether one path can produce result.
// Boolean atoms stay opaque because nil analysis does not solve them.
func (c *nilCondition) possible(result, nonNil bool) bool {
	switch c.kind {
	case 0:
		return result == !nonNil
	case 1:
		return result == nonNil
	case 2, 3, 4, 5:
		return true
	case 6:
		if result {
			return c.left.possible(true, nonNil) &&
				c.right.possible(true, nonNil)
		}
		return c.left.possible(false, nonNil) ||
			c.right.possible(false, nonNil)
	case 7:
		if result {
			return c.left.possible(true, nonNil) ||
				c.right.possible(true, nonNil)
		}
		return c.left.possible(false, nonNil) &&
			c.right.possible(false, nonNil)
	default:
		return c.left.possible(!result, nonNil)
	}
}
