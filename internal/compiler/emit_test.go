package compiler

import (
	"strings"
	"testing"
)

func TestPayloadFreeEnumEmitsGobMethods(t *testing.T) {
	t.Parallel()
	declaration := &model{
		Name: "Signal",
		Variants: []variant{
			{Name: "On"},
			{Name: "Off"},
		},
	}
	output := enumGo("signal.tgo", declaration, "fmt")
	for _, text := range []string{
		"func (v Signal) GobEncode() ([]byte, error)",
		"func (v *Signal) GobDecode(data []byte) error",
		"v.tgoTag < SignalTagOn || v.tgoTag > SignalTagOff",
	} {
		if !strings.Contains(output, text) {
			t.Fatalf("generated enum does not contain %q", text)
		}
	}
}

func TestPayloadEnumDoesNotEmitGobMethods(t *testing.T) {
	t.Parallel()
	declaration := &model{
		Name: "Message",
		Variants: []variant{{
			Name:   "Text",
			Fields: []field{{Name: "Body", Type: "string"}},
		}},
	}
	output := enumGo("message.tgo", declaration, "fmt")
	if strings.Contains(output, "GobEncode") || strings.Contains(output, "GobDecode") {
		t.Fatal("payload enum has gob methods")
	}
}
