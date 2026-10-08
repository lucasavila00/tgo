package compiler

import (
	"fmt"
	"go/parser"
	"go/token"
	"slices"
	"strings"
)

// sourceParser reads tgo extensions. The Go parser reads all other syntax.
type sourceParser struct {
	name   string
	input  string
	tokens []lexeme
	cursor int
	edits  []edit
	models []*model
}

func parseSource(files *token.FileSet, name string, data []byte) (*source, error) {
	tokens, err := lex(name, string(data))
	if err != nil {
		return nil, err
	}
	reader := sourceParser{name: name, input: string(data), tokens: tokens}
	if err := reader.declarations(); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	input := applyEdits(reader.input, reader.edits)
	input, err = rewriteExpressions(name, input)
	if err != nil {
		return nil, err
	}
	mode := parser.ParseComments | parser.AllErrors | parser.SkipObjectResolution
	file, err := parser.ParseFile(files, name, input, mode)
	if err != nil {
		return nil, err
	}
	return &source{Name: name, File: file, Models: reader.models}, nil
}

func (p *sourceParser) has(offset int, kind token.Token) bool {
	index := p.cursor + offset
	return index < len(p.tokens) && p.tokens[index].kind == kind
}

func (p *sourceParser) text(start, end int) string {
	if start == end {
		return ""
	}
	return p.input[p.tokens[start].start:p.tokens[end-1].end]
}

func (p *sourceParser) declarations() error {
	for p.cursor < len(p.tokens) {
		if p.has(0, token.TYPE) && p.has(1, token.IDENT) {
			handled, err := p.typeDeclaration()
			if err != nil {
				return err
			}
			if handled {
				continue
			}
		}
		if err := p.advance(); err != nil {
			return err
		}
	}
	return nil
}

// advance skips a nested group so its contents do not become declarations.
func (p *sourceParser) advance() error {
	if opening(p.tokens[p.cursor].kind) {
		end, err := closeToken(p.tokens, p.cursor)
		if err != nil {
			return err
		}
		p.cursor = end
	}
	p.cursor++
	return nil
}

func opening(kind token.Token) bool {
	return kind == token.LBRACE || kind == token.LBRACK || kind == token.LPAREN
}

func (p *sourceParser) typeDeclaration() (bool, error) {
	if p.cursor+2 >= len(p.tokens) {
		return false, nil
	}
	start := p.cursor
	declaration := &model{Name: p.tokens[start+1].text}
	p.cursor += 2
	var replacement string
	var err error
	switch {
	case p.tokens[p.cursor].text == "enum":
		replacement, err = p.enumDeclaration(declaration)
	case p.has(0, token.STRUCT) && p.has(1, token.LBRACE):
		replacement, err = p.structDeclaration(declaration)
	default:
		replacement, err = p.checkedDeclaration(declaration)
	}
	if err != nil {
		return false, err
	}
	if replacement == "" {
		p.cursor = start
		return false, nil
	}
	p.edits = append(p.edits, edit{
		start: p.tokens[start].start,
		end:   p.tokens[p.cursor-1].end,
		text:  replacement,
	})
	p.models = append(p.models, declaration)
	return true, nil
}

func (p *sourceParser) enumDeclaration(declaration *model) (string, error) {
	p.cursor++
	if !p.has(0, token.LBRACE) {
		return "", fmt.Errorf("enum %s needs a body", declaration.Name)
	}
	end, err := closeToken(p.tokens, p.cursor)
	if err != nil {
		return "", err
	}
	p.cursor++
	names := make(map[string]bool)
	for p.cursor < end {
		if p.has(0, token.SEMICOLON) {
			p.cursor++
			continue
		}
		item, err := p.variant()
		if err != nil {
			return "", err
		}
		if names[item.Name] {
			return "", fmt.Errorf("duplicate variant %s", item.Name)
		}
		names[item.Name] = true
		declaration.Variants = append(declaration.Variants, item)
	}
	if len(declaration.Variants) == 0 {
		return "", fmt.Errorf("enum %s has no variants", declaration.Name)
	}
	p.cursor = end + 1
	return enumGo(declaration), nil
}

func (p *sourceParser) variant() (variant, error) {
	if !p.has(0, token.IDENT) || !p.has(1, token.STRUCT) || !p.has(2, token.LBRACE) {
		return variant{}, fmt.Errorf("variant needs Name struct { fields }")
	}
	name := p.tokens[p.cursor].text
	p.cursor += 2
	fields, err := p.fields()
	if err != nil {
		return variant{}, err
	}
	return variant{Name: name, Fields: fields}, nil
}

func (p *sourceParser) structDeclaration(declaration *model) (string, error) {
	p.cursor++
	fields, err := p.fields()
	if err != nil {
		return "", err
	}
	declaration.Fields = fields
	text := "type " + declaration.Name + " struct {\n" + fieldDecls(fields) + "}"
	return text, nil
}

func (p *sourceParser) checkedDeclaration(declaration *model) (string, error) {
	start := p.cursor
	for p.cursor < len(p.tokens) && !p.has(0, token.SEMICOLON) {
		if p.tokens[p.cursor].text == "where" {
			declaration.Base = p.text(start, p.cursor)
			p.cursor++
			predicate := p.cursor
			if err := p.toSemicolon(); err != nil {
				return "", err
			}
			declaration.Predicate = p.text(predicate, p.cursor)
			if declaration.Base == "" || declaration.Predicate == "" {
				return "", fmt.Errorf("checked type needs a base type and predicate")
			}
			return checkedGo(declaration), nil
		}
		if err := p.advance(); err != nil {
			return "", err
		}
	}
	return "", nil
}

func (p *sourceParser) toSemicolon() error {
	for p.cursor < len(p.tokens) && !p.has(0, token.SEMICOLON) {
		if err := p.advance(); err != nil {
			return err
		}
	}
	return nil
}

func rewriteExpressions(name, input string) (string, error) {
	tokens, err := lex(name, input)
	if err != nil {
		return "", err
	}
	reader := sourceParser{name: name, input: input, tokens: tokens}
	for reader.cursor < len(tokens) {
		switch {
		case reader.tokens[reader.cursor].text == "match":
			if err := reader.match(); err != nil {
				return "", fmt.Errorf("%s: %w", name, err)
			}
		case reader.defaultMarker():
			reader.edits = append(reader.edits, edit{
				start: tokens[reader.cursor].start,
				end:   tokens[reader.cursor+2].end,
				text:  "__tgo_defaults: true",
			})
			reader.cursor += 3
		default:
			reader.cursor++
		}
	}
	return applyEdits(input, reader.edits), nil
}

func (p *sourceParser) match() error {
	start := p.cursor
	p.cursor++
	for p.cursor < len(p.tokens) && !p.has(0, token.LBRACE) {
		if err := p.advance(); err != nil {
			return err
		}
	}
	if !p.has(0, token.LBRACE) {
		return fmt.Errorf("match needs cases")
	}
	p.edits = append(p.edits,
		edit{start: p.tokens[start].start, end: p.tokens[start].end, text: "switch __tgo_match("},
		edit{start: p.tokens[p.cursor].start, end: p.tokens[p.cursor].start, text: ") "},
	)
	p.cursor++
	return nil
}

func applyEdits(input string, edits []edit) string {
	slices.SortStableFunc(edits, func(left, right edit) int { return left.start - right.start })
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

func (p *sourceParser) defaultMarker() bool {
	return p.has(0, token.PERIOD) && p.has(1, token.PERIOD) && p.has(2, token.DEFAULT)
}
