package tgolint

func noGenericValue() genericValue {
	return genericValue{
		function: nil, fact: nil, receiverArguments: nil, typeArguments: nil,
		conditionCall: nil, callDepth: 0, conditions: nil, maySkip: false,
	}
}

func genericFactHasEffectsAtOrAfter(fact *GenericEffectFact, callDepth int) bool {
	return len(genericFactEffectsAtOrAfter(fact, callDepth, true)) != 0 ||
		len(genericFactEffectsAtOrAfter(fact, callDepth, false)) != 0
}

func genericFactEffectsAtDepth(
	fact *GenericEffectFact,
	callDepth int,
	zero bool,
) []GenericEffect {
	if fact == nil {
		return nil
	}
	if callDepth == 0 {
		if zero {
			return fact.ZeroEffects
		}
		return fact.AccessEffects
	}
	if callDepth == 1 {
		if zero {
			return fact.ReturnedZeroEffects
		}
		return fact.ReturnedAccessEffects
	}
	result := []GenericEffect(nil)
	items := fact.ReturnedAccessEffectsAtDepth
	if zero {
		items = fact.ReturnedZeroEffectsAtDepth
	}
	for _, item := range items {
		if item.CallDepth == callDepth {
			result = append(result, item.Effect)
		}
	}
	return result
}

func genericFactEffectsAtOrAfter(
	fact *GenericEffectFact,
	callDepth int,
	zero bool,
) []GenericEffect {
	result := append([]GenericEffect(nil),
		genericFactEffectsAtDepth(fact, callDepth, zero)...,
	)
	if fact == nil {
		return result
	}
	items := fact.ReturnedAccessEffectsAtDepth
	if zero {
		items = fact.ReturnedZeroEffectsAtDepth
	}
	for _, item := range items {
		if item.CallDepth > callDepth {
			result = append(result, item.Effect)
		}
	}
	if callDepth == 0 {
		if zero {
			result = append(result, fact.ReturnedZeroEffects...)
		} else {
			result = append(result, fact.ReturnedAccessEffects...)
		}
	}
	return result
}
