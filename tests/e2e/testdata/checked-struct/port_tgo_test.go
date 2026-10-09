package checkedstruct

import (
	"errors"
	"strings"
	"testing"
)

func TestCheckedStructConstruction(t *testing.T) {
	port, err := NewPort(8080)
	if err != nil {
		t.Fatal(err)
		return
	}
	if Number(port) != 8080 {
		t.Fatalf("NewPort = %d, %v", Number(port), err)
	}
	_, err = NewPort(0)
	if !errors.Is(err, ErrInvalidPort) {
		t.Fatalf("NewPort error = %v", err)
	}
}

func TestNestedCheckedStructConstruction(t *testing.T) {
	port, err := NewServicePort(443)
	if err != nil {
		t.Fatal(err)
		return
	}
	if ServiceNumber(port) != 443 {
		t.Fatalf("NewServicePort = %d, %v", ServiceNumber(port), err)
	}
	_, err = NewServicePort(0)
	if !errors.Is(err, ErrInvalidPort) || !strings.Contains(err.Error(), "Port: ") {
		t.Fatalf("NewServicePort error = %v", err)
	}
}
