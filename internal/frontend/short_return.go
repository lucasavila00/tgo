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
	for returnIndex := len(tokens) - 1; returnIndex >= 0; returnIndex-- {
		if tokens[returnIndex].kind != token.RETURN {
			continue
		}
		operatorOffset, returnEnd, err := s.shortReturnSpan(tokens, returnIndex)
		if err != nil {
			return false, fmt.Errorf("%s: %w", original.Position(original.Pos(s.offsets[operatorOffset])), err)
		}
		if operatorOffset < 0 {
			continue
		}
		if tokens[returnIndex+1].offset == operatorOffset {
			returnEnd = s.shortReturnCommentStart(tokens, returnEnd)
		}
		operandIndex := returnIndex + 1
		leadingOperator := tokens[operandIndex].offset == operatorOffset
		operandStart := tokens[operandIndex].offset
		operandEnd := returnEnd
		if leadingOperator {
			operandStart = operatorOffset + 2
		} else {
			operandEnd = operatorOffset
		}
		markerName := ""
		for markerNumber := 1; ; markerNumber++ {
			markerName = fmt.Sprintf("tgoShortReturn%d", markerNumber)
			if !used[markerName] {
				used[markerName] = true
				break
			}
		}
		position := original.Position(original.Pos(s.offsets[operatorOffset]))
		records[markerName] = ShortReturn{ZeroSuccess: leadingOperator, Position: position}
		markerPrefix, markerSuffix := markerName+"(", ");"
		replacementText := markerPrefix + s.text[operandStart:operandEnd] + markerSuffix
		replacementOffsets := insertedOffsets(len(markerPrefix), s.offsets[tokens[operandIndex].offset])
		replacementOffsets = append(replacementOffsets, s.offsets[operandStart:operandEnd]...)
		replacementOffsets = append(replacementOffsets, insertedOffsets(len(markerSuffix), s.offsets[returnEnd-1])...)
		s.replace(tokens[operandIndex].offset, returnEnd, replacementText, replacementOffsets)
		return true, nil
	}
	return false, nil
}

func (s *mappedSource) shortReturnSpan(tokens []sourceToken, returnIndex int) (int, int, error) {
	operatorOffset := -1
	returnDepth := tokens[returnIndex].depth
	leadingOperator := false
	for i := returnIndex + 1; i < len(tokens); i++ {
		currentToken := tokens[i]
		atStatementEnd := currentToken.depth == returnDepth && (currentToken.kind == token.SEMICOLON || currentToken.kind == token.RBRACE)
		if currentToken.kind == token.EOF || atStatementEnd {
			if leadingOperator && i == returnIndex+3 {
				return operatorOffset, 0, fmt.Errorf("short return requires an error operand")
			}
			return operatorOffset, currentToken.offset, nil
		}
		if currentToken.depth != returnDepth || currentToken.kind != token.PERIOD || i+1 >= len(tokens) {
			continue
		}
		secondDot := tokens[i+1]
		if secondDot.kind != token.PERIOD || secondDot.offset != currentToken.offset+1 {
			continue
		}
		if operatorOffset >= 0 {
			return currentToken.offset, 0, fmt.Errorf("short return permits .. only before or after its operands")
		}
		operatorOffset = currentToken.offset
		leadingOperator = i == returnIndex+1
		i++
		if leadingOperator {
			continue
		}
		nextToken := tokens[i+1]
		atNextStatement := nextToken.kind == token.EOF || nextToken.kind == token.SEMICOLON || nextToken.kind == token.RBRACE
		hasNewline := strings.Contains(s.text[secondDot.end:nextToken.offset], "\n")
		if atNextStatement || hasNewline {
			return operatorOffset, tokens[i].end, nil
		}
		return operatorOffset, 0, fmt.Errorf("short return permits .. only before or after its operands")
	}
	return operatorOffset, len(s.text), nil
}

// shortReturnCommentStart leaves a final line comment outside the marker call.
func (s *mappedSource) shortReturnCommentStart(tokens []sourceToken, returnEnd int) int {
	operandEnd := 0
	for _, currentToken := range tokens {
		if currentToken.offset >= returnEnd {
			break
		}
		operandEnd = currentToken.end
	}
	positions := token.NewFileSet()
	commentFile := positions.AddFile("", -1, returnEnd-operandEnd)
	var commentScanner scanner.Scanner
	commentScanner.Init(commentFile, []byte(s.text[operandEnd:returnEnd]), nil, scanner.ScanComments)
	for {
		position, tokenKind, literal := commentScanner.Scan()
		if tokenKind == token.COMMENT && strings.HasPrefix(literal, "//") {
			return operandEnd + commentFile.Offset(position)
		}
		if tokenKind == token.EOF {
			return returnEnd
		}
	}
}
