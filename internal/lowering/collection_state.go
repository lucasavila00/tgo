package lowering

import (
	"go/ast"

	"github.com/lucasavila00/tgo/internal/frontend"
)

// registerComprehension connects generated ranges and output calls in this function.
func (r *rewrite) registerComprehension(record *frontend.Comprehension, destination *collectionDestination) {
	if r.functionState.comprehensionRanges == nil {
		r.functionState.comprehensionRanges = make(map[*ast.RangeStmt]bool)
		r.functionState.collectionOutputs = make(map[string]*collectionDestination)
	}
	for _, loop := range record.Ranges {
		r.functionState.comprehensionRanges[loop] = true
	}
	r.functionState.collectionOutputs[record.Output] = destination
}

func (r *rewrite) collectionDestination(name string) *collectionDestination {
	return r.functionState.collectionOutputs[name]
}
