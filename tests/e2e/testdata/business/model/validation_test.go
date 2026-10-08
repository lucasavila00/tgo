package model

import (
	"strings"
	"testing"
)

func TestValidateRejectsInvalidCheckedValue(t *testing.T) {
	value := Quantity{value: 0}
	if _, err := ValidateQuantity(value); err == nil {
		t.Fatal("ValidateQuantity accepted a value that fails its predicate")
	}
}

func TestValidateRejectsInvalidEnumTag(t *testing.T) {
	value := Account{tgoTag: 99}
	_, err := ValidateAccount(value)
	if err == nil || !strings.Contains(err.Error(), "invalid Account tag") {
		t.Fatalf("ValidateAccount returned %v", err)
	}
}

func TestValidateRebuildsNestedValuesAndPreservesCycles(t *testing.T) {
	amount, err := NewQuantity(3)
	if err != nil {
		t.Fatal(err)
	}
	input := &ValidationNode{Amount: amount}
	input.Next = input
	input.Other = input
	input.Items = []Quantity{amount}
	input.Alias = input.Items
	input.Fixed = [1]Quantity{amount}
	input.Lookup = map[string]Quantity{"amount": amount}
	input.AliasLookup = input.Lookup
	input.Dynamic = input
	input.Numbers = make(chan int)
	input.Callback = func() int { return 3 }

	output, err := ValidateValidationNode(*input)
	if err != nil {
		t.Fatal(err)
	}
	if output.Next != output.Next.Next {
		t.Fatal("validation did not preserve the pointer cycle")
	}
	if output.Dynamic != output.Next {
		t.Fatal("validation did not preserve the dynamic alias")
	}
	if output.Next != output.Other {
		t.Fatal("validation did not preserve the pointer alias")
	}
	replacement := Quantity{value: 8}
	output.Items[0] = replacement
	if output.Alias[0].Value() != 8 {
		t.Fatal("validation did not preserve the slice alias")
	}
	output.Lookup["amount"] = replacement
	if output.AliasLookup["amount"].Value() != 8 {
		t.Fatal("validation did not preserve the map alias")
	}
	if output.Numbers != input.Numbers || output.Callback() != 3 {
		t.Fatal("validation did not share safe channel and function values")
	}

	input.Items[0] = Quantity{value: 9}
	input.Lookup["amount"] = Quantity{value: 9}
	input.Next.Amount = Quantity{value: 9}
	if output.Items[0].Value() != 8 || output.Lookup["amount"].Value() != 8 ||
		output.Next.Amount.Value() != 3 {
		t.Fatal("the rebuilt graph changed after the input graph changed")
	}
}
