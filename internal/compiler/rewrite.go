package compiler

import (
	"fmt"
	"go/ast"
	"go/token"
	"slices"
	"strings"
)

// rewriteExpressions makes match and default markers valid Go syntax.
func rewriteExpressions(name, input string) (string, error) {
	tokens, err := lex(name, input)
	if err != nil {
		return "", err
	}

	parser := sourceParser{name: name, input: input, tokens: tokens}
	for parser.cursor < len(tokens) {
		switch {
		case parser.currentText() == "match":
			if err := parser.rewriteMatch(); err != nil {
				return "", fmt.Errorf("%s:%w", name, err)
			}
		case parser.atDefaultMarker():
			parser.rewriteDefaultMarker()
		default:
			parser.cursor++
		}
	}

	return applyEdits(input, parser.edits), nil
}

// currentText returns the current token text or an empty string at the end.
func (p *sourceParser) currentText() string {
	if p.cursor >= len(p.tokens) {
		return ""
	}
	return p.tokens[p.cursor].text
}

// rewriteMatch changes one match header into a temporary Go switch header.
func (p *sourceParser) rewriteMatch() error {
	keyword := p.cursor
	p.cursor++

	for p.cursor < len(p.tokens) && !p.has(0, token.LBRACE) {
		if err := p.advance(); err != nil {
			return err
		}
	}
	if !p.has(0, token.LBRACE) {
		return p.errorAt(keyword, "match needs cases")
	}

	openingBrace := p.tokens[p.cursor]
	p.edits = append(p.edits,
		edit{
			start: p.tokens[keyword].start,
			end:   p.tokens[keyword].end,
			text:  "switch __tgo_match(",
		},
		edit{
			start: openingBrace.start,
			end:   openingBrace.start,
			text:  ") ",
		},
	)
	p.cursor++
	return nil
}

// atDefaultMarker recognizes the three tokens in a default marker.
func (p *sourceParser) atDefaultMarker() bool {
	return p.has(0, token.PERIOD) &&
		p.has(1, token.PERIOD) &&
		p.has(2, token.DEFAULT)
}

// rewriteDefaultMarker changes one marker into a temporary keyed field.
func (p *sourceParser) rewriteDefaultMarker() {
	p.edits = append(p.edits, edit{
		start: p.tokens[p.cursor].start,
		end:   p.tokens[p.cursor+2].end,
		text:  "__tgo_defaults: true",
	})
	p.cursor += 3
}

// applyEdits applies source replacements from left to right.
func applyEdits(input string, edits []edit) string {
	slices.SortStableFunc(edits, compareEdits)

	var output strings.Builder
	previous := 0
	for _, change := range edits {
		output.WriteString(input[previous:change.start])
		output.WriteString(change.text)
		previous = change.end
	}
	output.WriteString(input[previous:])
	return output.String()
}

// compareEdits orders source edits by byte offset.
func compareEdits(left, right edit) int {
	return left.start - right.start
}

// removeLineDirectives keeps source maps out of generated Go output.
func removeLineDirectives(file *ast.File) {
	groups := file.Comments[:0]
	for _, group := range file.Comments {
		comments := group.List[:0]
		for _, comment := range group.List {
			lineComment := strings.HasPrefix(comment.Text, "//line ")
			blockComment := strings.HasPrefix(comment.Text, "/*line ")
			if !lineComment && !blockComment {
				comments = append(comments, comment)
			}
		}
		group.List = comments
		if len(group.List) > 0 {
			groups = append(groups, group)
		}
	}
	file.Comments = groups
}
