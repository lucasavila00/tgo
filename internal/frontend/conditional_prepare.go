package frontend

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"sort"
	"strings"
)

type sourceToken struct {
	offset int
	end    int
	kind   token.Token
	depth  int
}

func sourceTokens(source string) []sourceToken {
	set := token.NewFileSet()
	file := set.AddFile("", -1, len(source))
	var scan scanner.Scanner
	scan.Init(file, []byte(source), nil, 0)
	var tokens []sourceToken
	depth := 0
	for {
		position, kind, literal := scan.Scan()
		width := len(literal)
		if width == 0 {
			width = len(kind.String())
		}
		tokens = append(tokens, sourceToken{offset: file.Offset(position), end: file.Offset(position) + width, kind: kind, depth: depth})
		switch kind {
		case token.LPAREN, token.LBRACK, token.LBRACE:
			depth++
		case token.RPAREN, token.RBRACK, token.RBRACE:
			depth--
		}
		if kind == token.EOF {
			return tokens
		}
	}
}

// conditionalEnd asks Go's parser to confirm complete branch syntax.
// Balanced delimiters limit each parse to one possible conditional span.
func conditionalEnd(filename, source string, start int, tokens []sourceToken) int {
	const prefix = "package p; func f() { "
	braces, parens, brackets := 0, 0, 0
	first := sort.Search(len(tokens), func(i int) bool { return tokens[i].offset >= start })
	for i := first; i < len(tokens)-1; i++ {
		switch tokens[i].kind {
		case token.LPAREN:
			parens++
		case token.RPAREN:
			parens--
		case token.LBRACK:
			brackets++
		case token.RBRACK:
			brackets--
		case token.LBRACE:
			braces++
		case token.RBRACE:
			braces--
		}
		if tokens[i].kind != token.RBRACE || braces != 0 || parens != 0 || brackets != 0 || tokens[i+1].kind == token.ELSE {
			continue
		}
		end := tokens[i].offset + 1
		set := token.NewFileSet()
		parsed, err := parser.ParseFile(set, filename, prefix+source[start:end]+"\n}", parser.SkipObjectResolution)
		if err != nil {
			continue
		}
		if len(parsed.Decls) == 1 {
			if function, ok := parsed.Decls[0].(*ast.FuncDecl); ok && len(function.Body.List) == 1 {
				if conditional, ok := function.Body.List[0].(*ast.IfStmt); ok && set.Position(conditional.End()).Offset-len(prefix) == end-start {
					return end
				}
			}
		}
	}
	return -1
}

type conditionalEdit struct {
	start, end int
	name       string
}

// prepareConditionals wraps independent operands together. A nested operand
// is prepared before its containing expression; ambiguous syntax uses AST recovery.
func (s *mappedSource) prepareConditionals(filename string, original *token.File, used map[string]bool, conditionals map[string]token.Position) bool {
	var starts []int
	for offset, operand := range expressionIfTokens(s.text) {
		if operand {
			starts = append(starts, offset)
		}
	}
	if len(starts) == 0 {
		return false
	}
	sort.Sort(sort.Reverse(sort.IntSlice(starts)))
	tokens := sourceTokens(s.text)
	var edits []conditionalEdit
	nextName := 1
	depth := -1
	for _, start := range starts {
		index := sort.Search(len(tokens), func(i int) bool { return tokens[i].offset >= start })
		if tokens[index].depth < depth {
			continue
		}
		end := conditionalEnd(filename, s.text, start, tokens)
		if end < 0 || len(edits) > 0 && end > edits[len(edits)-1].start {
			continue
		}
		name := ""
		for {
			name = fmt.Sprintf("tgoConditional%d", nextName)
			nextName++
			if !used[name] {
				used[name] = true
				break
			}
		}
		conditionals[name] = original.Position(original.Pos(s.offsets[start]))
		edits = append(edits, conditionalEdit{start, end, name})
		depth = tokens[index].depth
	}
	if len(edits) == 0 {
		return false
	}
	var text strings.Builder
	offsets := make([]int, 0, len(s.offsets)+len(edits)*32)
	previous := 0
	for i := len(edits) - 1; i >= 0; i-- {
		edit := edits[i]
		before, after := edit.name+"(func(){", "})"
		text.WriteString(s.text[previous:edit.start])
		offsets = append(offsets, s.offsets[previous:edit.start]...)
		text.WriteString(before)
		offsets = append(offsets, insertedOffsets(len(before), s.offsets[edit.start])...)
		text.WriteString(s.text[edit.start:edit.end])
		offsets = append(offsets, s.offsets[edit.start:edit.end]...)
		text.WriteString(after)
		offsets = append(offsets, insertedOffsets(len(after), s.offsets[edit.end-1])...)
		previous = edit.end
	}
	text.WriteString(s.text[previous:])
	offsets = append(offsets, s.offsets[previous:]...)
	s.text, s.offsets = text.String(), offsets
	return true
}
