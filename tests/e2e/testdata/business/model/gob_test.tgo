package model

import (
	"bytes"
	"encoding/gob"
	"testing"
)

var gobDataSink []byte
var gobSignalSink Signal

func TestPayloadFreeEnumGobRoundTrip(t *testing.T) {
	t.Parallel()
	want := NewSignalOn()
	var data bytes.Buffer
	if err := gob.NewEncoder(&data).Encode(struct{ Kind Signal }{Kind: want}); err != nil {
		t.Fatal(err)
	}
	var fact struct{ Kind Signal }
	if err := gob.NewDecoder(&data).Decode(&fact); err != nil {
		t.Fatal(err)
	}
	if fact.Kind.Tag() != want.Tag() {
		t.Fatalf("decoded tag = %v, want %v", fact.Kind.Tag(), want.Tag())
	}
}

func TestPayloadFreeEnumGobWire(t *testing.T) {
	t.Parallel()
	value := NewSignalOff()
	data, err := value.GobEncode()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, []byte{0, 0, 0, 2}) {
		t.Fatalf("GobEncode() = %v, want [0 0 0 2]", data)
	}
	var decoded Signal
	if err := decoded.GobDecode(data); err != nil {
		t.Fatal(err)
	}
	if decoded.Tag() != SignalTagOff {
		t.Fatalf("decoded tag = %v, want %v", decoded.Tag(), SignalTagOff)
	}
	if err := decoded.GobDecode([]byte{0, 0, 0, 3}); err == nil {
		t.Fatal("GobDecode accepted an unknown tag")
	}
	if err := decoded.GobDecode([]byte{0, 0, 1, 1}); err == nil {
		t.Fatal("GobDecode accepted a truncated tag")
	}
	if decoded.Tag() != SignalTagOff {
		t.Fatal("failed GobDecode changed the receiver")
	}
}

func TestPayloadFreeEnumGobAllocations(t *testing.T) {
	value := NewSignalOn()
	encodeAllocations := testing.AllocsPerRun(1000, func() {
		data, err := value.GobEncode()
		if err != nil {
			panic(err)
		}
		gobDataSink = data
	})
	if encodeAllocations != 1 {
		t.Fatalf("GobEncode allocations = %v, want 1", encodeAllocations)
	}
	data := []byte{0, 0, 0, 1}
	decodeAllocations := testing.AllocsPerRun(1000, func() {
		if err := gobSignalSink.GobDecode(data); err != nil {
			panic(err)
		}
	})
	if decodeAllocations != 0 {
		t.Fatalf("GobDecode allocations = %v, want 0", decodeAllocations)
	}
}
