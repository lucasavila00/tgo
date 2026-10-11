package frontend

import (
	"fmt"
	"go/scanner"
	"go/token"
	"strings"
)

// ShortReturn records which results a return statement omits.
type ShortReturn struct {
	ZeroSuccess bool
	Position    token.Position
}

// prepareShortReturn wraps one operand list without adding a source function.
func (s *mappedSource) prepareShortReturn(original *token.File, used map[string]bool, records map[string]ShortReturn) (bool, error) {
	tokens := sourceTokens(s.text)
	for i := len(tokens) - 1; i >= 0; i-- {
		if tokens[i].kind != token.RETURN {
			continue
		}
		dots, end, err := s.shortReturnSpan(tokens, i)
		if err != nil {
			return false, fmt.Errorf("%s: %w", original.Position(original.Pos(s.offsets[dots])), err)
		}
		if dots < 0 {
			continue
		}
		if tokens[i+1].offset == dots {
			end = s.shortReturnCommentStart(tokens, end)
		}
		first := i + 1
		leading := tokens[first].offset == dots
		start := tokens[first].offset
		operandEnd := end
		if leading {
			start = dots + 2
		} else {
			operandEnd = dots
		}
		name := ""
		for next := 1; ; next++ {
			name = fmt.Sprintf("tgoShortReturn%d", next)
			if !used[name] {
				used[name] = true
				break
			}
		}
		position := original.Position(original.Pos(s.offsets[dots]))
		records[name] = ShortReturn{ZeroSuccess: leading, Position: position}
		before, after := name+"(", ");"
		text := before + s.text[start:operandEnd] + after
		offsets := insertedOffsets(len(before), s.offsets[tokens[first].offset])
		offsets = append(offsets, s.offsets[start:operandEnd]...)
		offsets = append(offsets, insertedOffsets(len(after), s.offsets[end-1])...)
		s.replace(tokens[first].offset, end, text, offsets)
		return true, nil
	}
	return false, nil
}

func (s *mappedSource) shortReturnSpan(tokens []sourceToken, start int) (int, int, error) {
	dots := -1
	depth := tokens[start].depth
	leading := false
	for i := start + 1; i < len(tokens); i++ {
		tok := tokens[i]
		if tok.kind == token.EOF || tok.depth == depth && (tok.kind == token.SEMICOLON || tok.kind == token.RBRACE) {
			if leading && i == start+3 {
				return dots, 0, fmt.Errorf("short return requires an error operand")
			}
			return dots, tok.offset, nil
		}
		if tok.depth != depth || tok.kind != token.PERIOD || i+1 >= len(tokens) || tokens[i+1].kind != token.PERIOD || tokens[i+1].offset != tok.offset+1 {
			continue
		}
		if dots >= 0 {
			return tok.offset, 0, fmt.Errorf("short return permits .. only before or after its operands")
		}
		dots = tok.offset
		leading = i == start+1
		i++
		if leading {
			continue
		}
		next := tokens[i+1]
		if next.kind == token.EOF || next.kind == token.SEMICOLON || next.kind == token.RBRACE || strings.Contains(s.text[tokens[i].end:next.offset], "\n") {
			return dots, tokens[i].end, nil
		}
		return dots, 0, fmt.Errorf("short return permits .. only before or after its operands")
	}
	return dots, len(s.text), nil
}

// shortReturnCommentStart leaves a final line comment outside the marker call.
func (s *mappedSource) shortReturnCommentStart(tokens []sourceToken, end int) int {
	last := 0
	for _, tok := range tokens {
		if tok.offset >= end {
			break
		}
		last = tok.end
	}
	set := token.NewFileSet()
	file := set.AddFile("", -1, end-last)
	var scan scanner.Scanner
	scan.Init(file, []byte(s.text[last:end]), nil, scanner.ScanComments)
	for {
		pos, kind, literal := scan.Scan()
		if kind == token.COMMENT && strings.HasPrefix(literal, "//") {
			return last + file.Offset(pos)
		}
		if kind == token.EOF {
			return end
		}
	}
}
