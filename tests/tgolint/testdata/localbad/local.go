package localbad

type countRepresentation struct {
	value int
}

type embedded struct {
	Event
}

func ReadFields(count Count, event Event) int {
	_ = count.value
	_ = event.tgoTag
	_ = event.tgoStarted
	return 0
}

func Convert(value countRepresentation) Count {
	return Count(value)
}

func BypassPortCheck() Port {
	return Port{number: 80}
}

func ReadPromoted(value embedded) EventTag {
	return value.tgoTag
}
