package frontend

import (
	"fmt"
	"strings"
)

// mappedSource owns prepared text and the position of each byte in the input.
// Syntax preparation supplies edits; only apply changes text and positions.
type mappedSource struct {
	input   string
	text    string
	offsets []int
}

// sourceEdit describes one replacement in the current prepared text.
// offsets maps replacement bytes to input bytes; the original end is retained.
type sourceEdit struct {
	start, end int
	text       string
	offsets    []int
}

func originalSource(input []byte) mappedSource {
	source := mappedSource{input: string(input), text: string(input), offsets: make([]int, len(input)+1)}
	for i := range source.offsets {
		source.offsets[i] = i
	}
	return source
}

// apply accepts edits in source order. It builds text and positions together.
func (s *mappedSource) apply(edits ...sourceEdit) {
	if len(edits) == 0 {
		return
	}
	var text strings.Builder
	offsets := make([]int, 0, len(s.offsets))
	previous := 0
	for _, edit := range edits {
		text.WriteString(s.text[previous:edit.start])
		offsets = append(offsets, s.offsets[previous:edit.start]...)
		text.WriteString(edit.text)
		offsets = append(offsets, edit.offsets...)
		previous = edit.end
	}
	text.WriteString(s.text[previous:])
	offsets = append(offsets, s.offsets[previous:]...)
	s.text, s.offsets = text.String(), offsets
}

func insertedOffsets(count, offset int) []int {
	result := make([]int, count)
	for i := range result {
		result[i] = offset
	}
	return result
}

// freshMarkerName reserves a name across all files in the compilation.
func freshMarkerName(used map[string]bool, prefix string) string {
	for number := 1; ; number++ {
		name := fmt.Sprintf("%s%d", prefix, number)
		if !used[name] {
			used[name] = true
			return name
		}
	}
}
