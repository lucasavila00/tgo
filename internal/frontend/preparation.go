package frontend

import (
	"go/ast"
	"go/token"
)

// sourcePreparation owns edits and TGo records for one file.
// Feature planners inspect text and return replacements; this owner applies them.
type sourcePreparation struct {
	source   mappedSource
	filename string
	original *token.File
	used     map[string]bool
	records  *Parsed
}

type preparedSyntax struct {
	file      *ast.File
	positions *token.File
}

// preparePostfix replaces postfix operators in the prepared source and records their meaning.
func (preparation *sourcePreparation) preparePostfix(sourceName string, input []byte) {
	replacements := postfixReplacements(sourceName, input, preparation.used)
	edits := make([]sourceEdit, len(replacements))
	for index, replacement := range replacements {
		edits[index] = replacement.edit
		preparation.records.Propagations[replacement.record.Name] = replacement.record
	}
	preparation.source.apply(edits...)
}

// prepareExpressions applies short-return, conditional, and comprehension edits until none remain.
// It updates source positions and feature records with each edit.
func (preparation *sourcePreparation) prepareExpressions(conditionals conditionalSyntax) error {
	for {
		shortReturn, err := preparation.source.shortReturn(preparation.original, preparation.used)
		if err != nil {
			return err
		}
		if shortReturn != nil {
			preparation.source.apply(shortReturn.edit)
			preparation.records.ShortReturns[shortReturn.name] = shortReturn.record
			continue
		}
		conditionalEdits := conditionals.prepare(preparation.source)
		preparation.applyConditionals(conditionalEdits)
		comprehension, err := preparation.source.comprehension(preparation.original,
			preparation.used)
		if err != nil {
			return err
		}
		if comprehension != nil {
			preparation.source.apply(comprehension.edit)
			preparation.records.Comprehensions[comprehension.name] = comprehension.record
		}
		if comprehension == nil && len(conditionalEdits) == 0 {
			return nil
		}
	}
}

// applyConditionals applies planned replacements and records each conditional at its input position.
func (preparation *sourcePreparation) applyConditionals(replacements []conditionalReplacement) {
	if len(replacements) == 0 {
		return
	}
	edits := make([]sourceEdit, len(replacements))
	for index, replacement := range replacements {
		edits[index] = replacement.edit
		preparation.records.Conditionals[replacement.name] = replacement.position
	}
	preparation.source.apply(edits...)
}
