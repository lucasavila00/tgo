package frontend

import (
	"go/scanner"
	"go/token"
)

// Marker records a call's postfix operator and original source span.
type Marker struct {
	Name     string
	Wrap     bool
	CallName string
	Position token.Position
	// EndOffset is the byte offset just after the operator in the input.
	EndOffset int
}

type postfixOperator struct {
	start, end int
	wrap       bool
}

type postfixReplacement struct {
	edit   sourceEdit
	record Marker
}

// postfixReplacements scans the input and plans a comment replacement for each postfix operator.
func postfixReplacements(filename string, input []byte, used map[string]bool) []postfixReplacement {
	fset := token.NewFileSet()
	file := fset.AddFile(filename, -1, len(input))
	var scan scanner.Scanner
	scan.Init(file, input, nil, scanner.ScanComments)
	var replacements []postfixReplacement
	previous := token.ILLEGAL
	for {
		pos, tok, _ := scan.Scan()
		if tok == token.EOF {
			break
		}
		if tok == token.COMMENT {
			continue
		}
		offset := file.Offset(pos)
		if tok == token.NOT && previous == token.RPAREN {
			operator := readPostfixOperator(&scan, input, offset)
			name := freshMarkerName(used, "tgoPropagation")
			text := "/*" + name + "*/"
			replacements = append(replacements, postfixReplacement{
				edit: sourceEdit{operator.start, operator.end, text, insertedOffsets(len(text), operator.start)},
				record: Marker{Name: name, Wrap: operator.wrap, Position: file.Position(pos),
					EndOffset: operator.end},
			})
			previous = token.ILLEGAL
			continue
		}
		previous = tok
	}
	return replacements
}

// readPostfixOperator records the operator's meaning and input span after its first !.
// It consumes the second ! only when the two bytes are adjacent.
func readPostfixOperator(scan *scanner.Scanner, input []byte, start int) postfixOperator {
	operator := postfixOperator{start: start, end: start + 1}
	if operator.end < len(input) && input[operator.end] == '!' {
		_, next, _ := scan.Scan()
		if next != token.NOT {
			panic("invalid propagation suffix")
		}
		operator.end++
		operator.wrap = true
	}
	return operator
}
