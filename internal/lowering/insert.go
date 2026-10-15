package lowering

import (
	"fmt"
	"github.com/lucasavila00/tgo/internal/lowering/shared"

	"github.com/lucasavila00/tgo/internal/lowering/source"
)

// InsertAfter returns Go files with the callback's statements after the selected expression.
// The callback receives saved results; the original syntax tree stays unchanged.
func (s *Source) InsertAfter(site Site, after After) (map[string][]byte, error) {
	target, ok := s.inventory.Expressions[source.Site(site)]
	if !ok {
		return nil, fmt.Errorf("expression site does not belong to this source")
	}
	if site.Reason != "" {
		return nil, fmt.Errorf("expression has no current-function evaluation: %s", site.Reason)
	}
	r := newRewrite(s, shared.Input{Target: target, After: shared.After(after)})
	output, err := r.Emit()
	if err != nil {
		return nil, err
	}
	if !output.Inserted {
		return nil, fmt.Errorf("expression site was not lowered: %s:%d", site.File, site.Start)
	}
	return output.Files, nil
}

// LowerPropagations replaces checked TGo operations with ordinary Go.
// It returns formatted file contents by absolute path without writing files.
// A package with no TGo syntax returns no changed files.
// It reports an error if any accepted operation was not replaced.
func (s *Source) LowerPropagations() (map[string][]byte, error) {
	r := newRewrite(s, shared.Input{TGo: true})
	hasWork := false
	for _, file := range s.inventory.Checked.Syntax {
		if r.Contains(file) {
			hasWork = true
			break
		}
	}
	if !hasWork {
		return nil, nil
	}
	output, err := r.Emit()
	if err != nil {
		return nil, err
	}
	if err := r.CheckLoweredOperations(); err != nil {
		return nil, err
	}
	return output.Files, nil
}
