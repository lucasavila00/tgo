package compiler

import (
	"fmt"
	"go/ast"
	"go/token"
	"slices"
	"strings"
)

type rewriteMarkers struct {
	match    string
	defaults string
}

// rewriteExpressions makes match and default markers valid Go syntax.
func rewriteExpressions(name, input string) (string, rewriteMarkers, error) {
	tokens, err := lex(name, input)
	if err != nil {
		return "", rewriteMarkers{}, err
	}

	used := identifierNames(tokens)
	markers := rewriteMarkers{
		match:    freshIdentifier("__tgo_match", used),
		defaults: freshIdentifier("__tgo_defaults", used),
	}
	parser := sourceParser{
		name:          name,
		input:         input,
		tokens:        tokens,
		matchMarker:   markers.match,
		defaultMarker: markers.defaults,
	}
	for parser.cursor < len(tokens) {
		match, err := parser.atMatchStatement()
		if err != nil {
			return "", rewriteMarkers{}, fmt.Errorf("%s:%w", name, err)
		}
		switch {
		case match:
			if err := parser.rewriteMatch(); err != nil {
				return "", rewriteMarkers{}, fmt.Errorf("%s:%w", name, err)
			}
		case parser.atDefaultMarker():
			parser.rewriteDefaultMarker()
		default:
			parser.cursor++
		}
	}

	return applyEdits(input, parser.edits), markers, nil
}

// atMatchStatement distinguishes tgo match from a Go identifier named match.
func (p *sourceParser) atMatchStatement() (bool, error) {
	if p.currentText() != "match" || !p.atStatementStart() {
		return false, nil
	}
	if p.has(1, token.COLON) {
		return false, nil
	}
	opening, err := p.matchBody(p.cursor + 1)
	if err != nil {
		return false, err
	}
	return opening >= 0, nil
}

// atStatementStart reports a token position that can start a Go statement.
func (p *sourceParser) atStatementStart() bool {
	if p.cursor == 0 {
		return false
	}
	previous := p.tokens[p.cursor-1].kind
	return previous == token.LBRACE || previous == token.SEMICOLON ||
		previous == token.COLON
}

// matchBody finds a case body before the end of the current statement.
func (p *sourceParser) matchBody(start int) (int, error) {
	for position := start; position < len(p.tokens); position++ {
		current := p.tokens[position].kind
		if current == token.SEMICOLON {
			if position == start && p.implicitSemicolon(position) {
				continue
			}
			return -1, nil
		}
		if !opening(current) {
			continue
		}
		end, err := closeToken(p.tokens, position)
		if err != nil {
			return -1, err
		}
		if current == token.LBRACE {
			first := p.tokens[position+1].kind
			caseBody := position+1 < end &&
				(first == token.CASE || first == token.DEFAULT)
			emptyBody := position+1 == end && p.groupEndsStatement(end)
			if caseBody || emptyBody {
				return position, nil
			}
		}
		position = end
	}
	return -1, nil
}

// groupEndsStatement reports a group followed by a semicolon or file end.
func (p *sourceParser) groupEndsStatement(end int) bool {
	return end+1 == len(p.tokens) || p.tokens[end+1].kind == token.SEMICOLON
}

// identifierNames gets every identifier spelling in one source file.
func identifierNames(tokens []lexeme) map[string]bool {
	names := make(map[string]bool)
	for _, item := range tokens {
		if item.kind == token.IDENT {
			names[item.text] = true
		}
	}
	return names
}

// freshIdentifier gets an internal name absent from source tokens.
func freshIdentifier(base string, used map[string]bool) string {
	name := base
	for suffix := 1; used[name]; suffix++ {
		name = fmt.Sprintf("%s_%d", base, suffix)
	}
	used[name] = true
	return name
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
	opening, err := p.matchBody(keyword + 1)
	if err != nil {
		return err
	}
	if opening < 0 {
		return p.errorAt(keyword, "match needs cases")
	}

	openingBrace := p.tokens[opening]
	p.edits = append(p.edits,
		edit{
			start: p.tokens[keyword].start,
			end:   p.tokens[keyword].end,
			text:  "switch " + p.matchMarker + "(",
		},
		edit{
			start: openingBrace.start,
			end:   openingBrace.start,
			text:  ") ",
		},
	)
	p.cursor = opening + 1
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
		text:  p.defaultMarker + ": true",
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
