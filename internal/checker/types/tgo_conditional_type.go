package types

// untypedConditionalType finds the common untyped branch type.
// It returns nil when there is no common type or a nil branch needs a supplied type.
func untypedConditionalType(values []*operand) Type {
	var typ Type
	for _, branch := range values {
		if branch.isNil() {
			continue
		}
		if typ == nil {
			typ = branch.typ()
		} else {
			typ = maxType(typ, branch.typ())
			if typ == nil {
				break
			}
		}
	}
	for _, branch := range values {
		if branch.isNil() && typ != nil {
			typ = nil
			break
		}
	}
	return typ
}
