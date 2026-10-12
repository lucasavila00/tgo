package types

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
