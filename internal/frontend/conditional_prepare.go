package frontend

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
)

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
		if tokens[i].kind != token.RBRACE || braces != 0 || parens != 0 || brackets != 0 || tokens[i+
			1].kind == token.ELSE {
			continue
		}
		end := tokens[i].offset + 1
		set := token.NewFileSet()
		parsed, err := parser.ParseFile(set, filename, prefix+source[start:end]+"\n}", parser.SkipObjectResolution)
		if err != nil {
			continue
		}
		if len(parsed.Decls) != 1 {
			continue
		}
		function, isFunction := parsed.Decls[0].(*ast.FuncDecl)
		if !isFunction || len(function.Body.List) != 1 {
			continue
		}
		conditional := function.Body.List[0]
		switch conditional.(type) {
		case *ast.IfStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt:
			parsedEnd := set.Position(conditional.End()).Offset - len(prefix)
			if parsedEnd == end-start {
				return end
			}
		}
	}
	return -1
}

type conditionalReplacement struct {
	edit     sourceEdit
	name     string
	position token.Position
}

// conditionalSyntax owns operand recognition for both preparation and parser recovery.
// Both paths produce the same replacement and source record.
type conditionalSyntax struct {
	filename string
	original *token.File
	used     map[string]bool
}

// prepareConditionals wraps independent operands together. A nested operand
// is prepared before its containing expression; ambiguous syntax uses AST recovery.
func (syntax conditionalSyntax) prepare(source mappedSource) []conditionalReplacement {
	s := &source
	var starts []int
	for offset, operand := range expressionConditionalTokens(s.text) {
		if operand {
			starts = append(starts, offset)
		}
	}
	if len(starts) == 0 {
		return nil
	}
	sort.Sort(sort.Reverse(sort.IntSlice(starts)))
	tokens := sourceTokens(s.text)
	var replacements []conditionalReplacement
	nextName := 1
	depth := -1
	for _, start := range starts {
		index := sort.Search(len(tokens), func(i int) bool { return tokens[i].offset >= start })
		if tokens[index].depth < depth {
			continue
		}
		end := conditionalEnd(syntax.filename, s.text, start, tokens)
		if end < 0 || len(replacements) > 0 && end > replacements[len(replacements)-1].edit.start {
			continue
		}
		name := ""
		for {
			name = fmt.Sprintf("tgoConditional%d", nextName)
			nextName++
			if !syntax.used[name] {
				syntax.used[name] = true
				break
			}
		}
		replacements = append(replacements, syntax.replacement(source, start, end, name))
		depth = tokens[index].depth
	}
	// Plans were found from right to left; application uses source order.
	for left, right := 0, len(replacements)-1; left < right; left, right = left+1, right-1 {
		replacements[left], replacements[right] = replacements[right], replacements[left]
	}
	return replacements
}

func (syntax conditionalSyntax) replacement(source mappedSource, start, end int, name string) conditionalReplacement {
	before, after := name+"(func(){", "})"
	offsets := insertedOffsets(len(before), source.offsets[start])
	offsets = append(offsets, source.offsets[start:end]...)
	offsets = append(offsets, insertedOffsets(len(after), source.offsets[end-1])...)
	return conditionalReplacement{
		edit: sourceEdit{start, end, before + source.text[start:end] + after, offsets},
		name: name, position: syntax.original.Position(syntax.original.Pos(source.offsets[start])),
	}
}
