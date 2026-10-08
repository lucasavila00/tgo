package compiler

import (
	"fmt"
	"go/parser"
	"go/token"
)

// sourceParser reads tgo extensions. The Go parser reads all other syntax.
type sourceParser struct {
	name          string
	input         string
	tokens        []lexeme
	cursor        int
	edits         []edit
	models        []*model
	matchMarker   string
	defaultMarker string
}

// parseSource lowers tgo syntax and parses the result as a Go file.
func parseSource(files *token.FileSet, name string, data []byte) (*source, error) {
	tokens, err := lex(name, string(data))
	if err != nil {
		return nil, err
	}
	reader := sourceParser{name: name, input: string(data), tokens: tokens}
	if err := reader.declarations(); err != nil {
		return nil, fmt.Errorf("%s:%w", name, err)
	}
	input := applyEdits(reader.input, reader.edits)
	input, markers, err := rewriteExpressions(name, input)
	if err != nil {
		return nil, err
	}
	mode := parser.ParseComments | parser.AllErrors | parser.SkipObjectResolution
	file, err := parser.ParseFile(files, name, input, mode)
	if err != nil {
		return nil, err
	}
	return &source{
		Name:          name,
		File:          file,
		Models:        reader.models,
		MatchMarker:   markers.match,
		DefaultMarker: markers.defaults,
	}, nil
}

// has reports whether a token at the current offset has the given kind.
func (p *sourceParser) has(offset int, kind token.Token) bool {
	index := p.cursor + offset
	return index < len(p.tokens) && p.tokens[index].kind == kind
}

// text returns source text for a half-open token range.
func (p *sourceParser) text(start, end int) string {
	if start == end {
		return ""
	}
	return p.input[p.tokens[start].start:p.tokens[end-1].end]
}

// errorAt reports a parser error at one original tgo token.
func (p *sourceParser) errorAt(index int, pattern string, args ...any) error {
	if index >= len(p.tokens) {
		return fmt.Errorf(pattern, args...)
	}
	message := fmt.Sprintf(pattern, args...)
	return fmt.Errorf("%d:%d: %s", p.tokens[index].line, p.tokens[index].column, message)
}

// declarations finds and lowers top-level tgo type declarations.
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

// opening reports whether a token starts a nested group.
func opening(kind token.Token) bool {
	return kind == token.LBRACE || kind == token.LBRACK || kind == token.LPAREN
}

// typeDeclaration parses an enum, checked type, or struct with defaults.
func (p *sourceParser) typeDeclaration() (bool, error) {
	if p.cursor+2 >= len(p.tokens) {
		return false, nil
	}
	start := p.cursor
	name := p.tokens[start+1]
	declaration := &model{Name: name.text, Line: name.line, Column: name.column}
	p.cursor += 2
	var replacement string
	var err error
	switch {
	case p.tokens[p.cursor].text == "enum" && p.enumBodyFollows():
		replacement, err = p.enumDeclaration(declaration)
	default:
		original := p.cursor
		replacement, err = p.checkedDeclaration(declaration)
		if replacement == "" && err == nil {
			p.cursor = original
			if p.has(0, token.STRUCT) && p.has(1, token.LBRACE) {
				replacement, err = p.structDeclaration(declaration)
			}
		}
	}
	if err != nil {
		return false, err
	}
	if replacement == "" {
		p.cursor = start
		return false, nil
	}
	if p.has(0, token.SEMICOLON) {
		p.cursor++
	}
	p.edits = append(p.edits, edit{
		start: p.tokens[start].start,
		end:   p.tokens[p.cursor-1].end,
		text: generatedSource(
			p.name,
			p.tokens[start].line,
			p.tokens[p.cursor-1].line,
			replacement,
		),
	})
	p.models = append(p.models, declaration)
	return true, nil
}

// enumBodyFollows accepts a body after an optional implicit semicolon.
func (p *sourceParser) enumBodyFollows() bool {
	if p.has(1, token.LBRACE) {
		return true
	}
	return p.implicitSemicolon(p.cursor+1) && p.has(2, token.LBRACE)
}

// generatedSource keeps generated declarations on their tgo source lines.
func generatedSource(name string, startLine, endLine int, code string) string {
	return fmt.Sprintf(
		"//line %s:%d:1\n%s\n//line %s:%d:1\n",
		name,
		startLine,
		code,
		name,
		endLine,
	)
}

// lineDirective maps the next generated line to one tgo source token.
func lineDirective(name string, line, column int) string {
	return fmt.Sprintf("//line %s:%d:%d\n", name, line, column)
}

// inlineLineDirective maps the token after it without adding a source line.
func inlineLineDirective(name string, line, column int) string {
	return fmt.Sprintf("/*line %s:%d:%d*/", name, line, column)
}

// enumDeclaration parses variants and emits their Go representation.
func (p *sourceParser) enumDeclaration(declaration *model) (string, error) {
	enumToken := p.cursor
	p.cursor++
	if p.implicitSemicolon(p.cursor) {
		p.cursor++
	}
	if !p.has(0, token.LBRACE) {
		return "", p.errorAt(enumToken, "enum %s needs a body", declaration.Name)
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
		variantToken := p.cursor
		item, err := p.variant()
		if err != nil {
			return "", err
		}
		if names[item.Name] {
			return "", p.errorAt(variantToken, "duplicate variant %s", item.Name)
		}
		names[item.Name] = true
		declaration.Variants = append(declaration.Variants, item)
	}
	if len(declaration.Variants) == 0 {
		return "", p.errorAt(enumToken, "enum %s has no variants", declaration.Name)
	}
	p.cursor = end + 1
	return enumGo(p.name, declaration), nil
}

// variant parses one named enum payload.
func (p *sourceParser) variant() (variant, error) {
	if !p.has(0, token.IDENT) || !p.has(1, token.STRUCT) || !p.has(2, token.LBRACE) {
		return variant{}, p.errorAt(p.cursor, "variant needs Name struct { fields }")
	}
	name := p.tokens[p.cursor].text
	p.cursor += 2
	fields, err := p.fields()
	if err != nil {
		return variant{}, err
	}
	return variant{Name: name, Fields: fields}, nil
}

// structDeclaration removes defaults from one tgo struct declaration.
func (p *sourceParser) structDeclaration(declaration *model) (string, error) {
	p.cursor++
	fields, err := p.fields()
	if err != nil {
		return "", err
	}
	declaration.Fields = fields
	text := "type " + declaration.Name + " struct {\n" + fieldDecls(p.name, fields) + "}"
	return text, nil
}

// checkedDeclaration parses a base type and its construction predicate.
func (p *sourceParser) checkedDeclaration(declaration *model) (string, error) {
	start := p.cursor
	for p.cursor < len(p.tokens) && !p.has(0, token.SEMICOLON) {
		if p.tokens[p.cursor].text != "where" ||
			!goTypeExpression(p.text(start, p.cursor)) {
			if err := p.advance(); err != nil {
				return "", err
			}
			continue
		}
		return p.finishCheckedDeclaration(start, declaration)
	}
	return "", nil
}

// finishCheckedDeclaration reads a checked predicate after its where keyword.
func (p *sourceParser) finishCheckedDeclaration(
	start int,
	declaration *model,
) (string, error) {
	declaration.Base = p.text(start, p.cursor)
	declaration.BaseLine = p.tokens[start].line
	declaration.BaseColumn = p.tokens[start].column
	p.cursor++
	if p.implicitSemicolon(p.cursor) {
		p.cursor++
	}
	predicate := p.cursor
	if predicate < len(p.tokens) {
		declaration.PredicateLine = p.tokens[predicate].line
		declaration.PredicateColumn = p.tokens[predicate].column
	}
	if err := p.toSemicolon(); err != nil {
		return "", err
	}
	declaration.Predicate = p.text(predicate, p.cursor)
	if declaration.Predicate == "" {
		return "", p.errorAt(start, "checked type needs a base type and predicate")
	}
	return checkedGo(p.name, declaration), nil
}

// implicitSemicolon reports a newline semicolon inserted by the Go scanner.
func (p *sourceParser) implicitSemicolon(index int) bool {
	return index < len(p.tokens) && p.tokens[index].kind == token.SEMICOLON &&
		p.tokens[index].start == p.tokens[index].end
}

// goTypeExpression reports syntax that can start a checked base type.
func goTypeExpression(text string) bool {
	if text == "" {
		return false
	}
	_, err := parser.ParseExpr(text)
	return err == nil
}

// toSemicolon advances across one top-level declaration expression.
func (p *sourceParser) toSemicolon() error {
	for p.cursor < len(p.tokens) && !p.has(0, token.SEMICOLON) {
		if err := p.advance(); err != nil {
			return err
		}
	}
	return nil
}
