package good

import "example.com/tgolint/model"

func UnusedPointerAlias(value *model.Event) string {
	alias := value
	_ = alias
	if value.Tag() == model.EventTagStarted {
		return value.StartedPayload().ID
	}
	return ""
}

func ReadOnlyPointerAlias(value *model.Event) string {
	alias := value
	if value.Tag() == model.EventTagStarted {
		_ = alias.Tag()
		return value.StartedPayload().ID
	}
	return ""
}

func ReboundPointerAlias(value, other *model.Event) string {
	alias := value
	alias = other
	if value.Tag() == model.EventTagStarted {
		*alias = model.NewEventStopped("changed")
		return value.StartedPayload().ID
	}
	return ""
}

func ShadowedPointerAlias(value, other *model.Event) string {
	alias := value
	_ = alias
	if value.Tag() == model.EventTagStarted {
		{
			alias := other
			*alias = model.NewEventStopped("changed")
		}
		return value.StartedPayload().ID
	}
	return ""
}

func UnusedCapturedPointerAlias(value *model.Event) string {
	alias := value
	unused := func() { _ = alias }
	_ = unused
	if value.Tag() == model.EventTagStarted {
		return value.StartedPayload().ID
	}
	return ""
}

func ReboundBeforePointerProof(value, other *model.Event) string {
	alias := value
	value = other
	if value.Tag() == model.EventTagStarted {
		*alias = model.NewEventStopped("changed")
		return value.StartedPayload().ID
	}
	return ""
}

type EventPointerEnvelope struct {
	Event *model.Event
}

func ReboundPointerFieldBeforeProof(
	envelope EventPointerEnvelope,
	other *model.Event,
) string {
	alias := envelope.Event
	envelope.Event = other
	if envelope.Event.Tag() == model.EventTagStarted {
		*alias = model.NewEventStopped("changed")
		return envelope.Event.StartedPayload().ID
	}
	return ""
}

func ReboundCapturedAlias(value, other *model.Event) string {
	alias := value
	mutate := func() { *alias = model.NewEventStopped("changed") }
	alias = other
	if value.Tag() == model.EventTagStarted {
		mutate()
		return value.StartedPayload().ID
	}
	return ""
}

func RetestedPointerAlias(value *model.Event) string {
	alias := value
	*alias = model.NewEventStarted("changed", "")
	if value.Tag() == model.EventTagStarted {
		return value.StartedPayload().ID
	}
	return ""
}

func InvokedReboundClosure(value, other *model.Event) string {
	alias := value
	mutate := func() { *alias = model.NewEventStopped("changed") }
	alias = other
	if value.Tag() == model.EventTagStarted {
		mutate()
		return value.StartedPayload().ID
	}
	return ""
}

func UntouchedBranchAlias(value, other *model.Event, change bool) string {
	alias := other
	if change {
		*alias = model.NewEventStopped("changed")
	}
	if value.Tag() == model.EventTagStarted {
		return value.StartedPayload().ID
	}
	return ""
}

func PayloadBeforeOperandMutation(value *model.Event) string {
	if value.Tag() == model.EventTagStarted {
		return value.StartedPayload().ID + mutateSafeAliasString(value)
	}
	return ""
}

func mutateSafeAliasString(value *model.Event) string {
	*value = model.NewEventStopped("changed")
	return ""
}

func ConditionMutationBeforeRetest(value *model.Event) string {
	if mutateSafeAliasString(value) == "" &&
		value.Tag() == model.EventTagStopped {
		return value.StoppedPayload().Reason
	}
	return ""
}

func SafeAssignmentOperandSnapshot(value, other *model.Event) string {
	alias := other
	rebind := func() *model.Event {
		alias = value
		return value
	}
	if value.Tag() == model.EventTagStarted {
		*saveGoodAliasPointer(alias), _ = model.NewEventStopped("changed"), rebind()
		return value.StartedPayload().ID
	}
	return ""
}

func saveGoodAliasPointer(value *model.Event) *model.Event { return value }

func RecursiveUnrelatedArgument(value, other, proved *model.Event) string {
	var mutate func(*model.Event, int)
	mutate = func(pointer *model.Event, depth int) {
		if depth == 0 {
			*pointer = model.NewEventStopped("changed")
			return
		}
		mutate(other, 0)
	}
	if proved.Tag() == model.EventTagStarted {
		mutate(value, 1)
		return proved.StartedPayload().ID
	}
	return ""
}

func NestedClosureUnrelatedAlias(value, other *model.Event) string {
	mutate := func() { *other = model.NewEventStopped("changed") }
	wrapper := func() { mutate() }
	if value.Tag() == model.EventTagStarted {
		wrapper()
		return value.StartedPayload().ID
	}
	return ""
}

type recursiveEventNode struct {
	Next  *recursiveEventNode
	Event *model.Event
}

func RecursiveFieldCopiedAlias(value *recursiveEventNode) string {
	event := value.Next.Event
	if event.Tag() == model.EventTagStarted {
		return event.StartedPayload().ID
	}
	return ""
}

func SummaryChangedOtherArgument(value, other *model.Event) string {
	mutate := func(pointer *model.Event) {
		*pointer = model.NewEventStopped("changed")
	}
	mutate(value)
	if value.Tag() != model.EventTagStarted {
		return ""
	}
	mutate(other)
	return value.StartedPayload().ID
}

func MutualRecursiveUnrelatedArgument(
	value, other, proved *model.Event,
) string {
	var first, second func(*model.Event, int)
	first = func(pointer *model.Event, depth int) {
		if depth == 0 {
			*pointer = model.NewEventStopped("changed")
			return
		}
		second(other, 0)
	}
	second = func(pointer *model.Event, depth int) { first(pointer, depth) }
	if proved.Tag() == model.EventTagStarted {
		first(value, 1)
		return proved.StartedPayload().ID
	}
	return ""
}
