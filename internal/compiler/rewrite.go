package compiler

import (
	"fmt"
	"go/token"
	"slices"
	"strings"
)

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
				return "", fmt.Errorf("%s: %w", name, err)
			}
		case parser.atDefaultMarker():
			parser.rewriteDefaultMarker()
		default:
			parser.cursor++
		}
	}

	return applyEdits(input, parser.edits), nil
}

func (p *sourceParser) currentText() string {
	if p.cursor >= len(p.tokens) {
		return ""
	}
	return p.tokens[p.cursor].text
}

func (p *sourceParser) rewriteMatch() error {
	keyword := p.cursor
	p.cursor++

	for p.cursor < len(p.tokens) && !p.has(0, token.LBRACE) {
		if err := p.advance(); err != nil {
			return err
		}
	}
	if !p.has(0, token.LBRACE) {
		return fmt.Errorf("match needs cases")
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

func (p *sourceParser) atDefaultMarker() bool {
	return p.has(0, token.PERIOD) &&
		p.has(1, token.PERIOD) &&
		p.has(2, token.DEFAULT)
}

func (p *sourceParser) rewriteDefaultMarker() {
	p.edits = append(p.edits, edit{
		start: p.tokens[p.cursor].start,
		end:   p.tokens[p.cursor+2].end,
		text:  "__tgo_defaults: true",
	})
	p.cursor += 3
}

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

func compareEdits(left, right edit) int {
	return left.start - right.start
}
