package types

import "go/token"

// enumTypeArgument checks that a constraint admits an enum through its declared storage family.
func (check *Checker) enumTypeArgument(position token.Pos, argument, bound Type) string {
	if check == nil || !check.nativeSource(position) {
		return ""
	}
	allowed := bound.Underlying().(*Interface).typeSet().terms
	for _, candidate := range enumArgumentTerms(argument) {
		if candidate == nil || !hasEnumStorage(candidate.typ) {
			continue
		}
		if !enumTermAdmitted(candidate, allowed) {
			return "enum type argument requires a constraint that preserves its declared storage type"
		}
	}
	return ""
}

// enumArgumentTerms returns the finite alternatives of a type parameter or one concrete type.
func enumArgumentTerms(argument Type) termlist {
	if parameter, ok := Unalias(argument).(*TypeParam); ok {
		return parameter.iface().typeSet().terms
	}
	return termlist{{typ: argument}}
}

// enumTermAdmitted accepts unconstrained use or a matching term with the same storage owner.
func enumTermAdmitted(candidate *term, allowed termlist) bool {
	for _, witness := range allowed {
		if witness == nil || !candidate.subsetOf(witness) {
			continue
		}
		if witness.typ == nil || sameEnumStorage(candidate.typ, witness.typ) {
			return true
		}
	}
	return false
}
