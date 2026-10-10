package bad

import (
	"testing"

	"example.com/tgolint/model"
)

func TestPointerAliasExamplesReachWrongPayload(t *testing.T) {
	tests := []struct {
		name string
		read func(*model.Event) string
	}{
		{name: "before if", read: AliasBeforeIf},
		{name: "after if", read: AliasAfterIf},
		{name: "before switch", read: AliasBeforeSwitch},
		{name: "invoked closure", read: InvokedAliasClosure},
		{name: "escaped closure", read: EscapedAliasClosure},
		{
			name: "pointer field after switch",
			read: func(value *model.Event) string {
				return PointerFieldAliasAfterSwitch(EventPointerEnvelope{Event: value})
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := model.NewEventStarted("original", "")
			if result := test.read(&value); result != "" {
				t.Fatalf("payload result = %q, want empty wrong payload", result)
			}
			if value.Tag() != model.EventTagStopped {
				t.Fatalf("tag = %d, want %d", value.Tag(), model.EventTagStopped)
			}
		})
	}
}
