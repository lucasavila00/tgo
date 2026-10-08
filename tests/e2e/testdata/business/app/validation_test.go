package app

import (
	"testing"

	"example.com/business/model"
)

func TestValidateCrossPackageCycle(t *testing.T) {
	amount, err := model.NewQuantity(4)
	if err != nil {
		t.Fatal(err)
	}
	node := &model.ValidationNode{Amount: amount}
	node.Items = []model.Quantity{amount}
	node.Alias = node.Items
	node.Fixed = [1]model.Quantity{amount}
	node.Lookup = map[string]model.Quantity{"amount": amount}
	node.AliasLookup = node.Lookup
	input := ValidationEnvelope{Root: node}
	node.Dynamic = &input

	output, err := ValidateValidationEnvelope(input)
	if err != nil {
		t.Fatal(err)
	}
	cycle, ok := output.Root.Dynamic.(*ValidationEnvelope)
	if !ok || cycle.Root != output.Root {
		t.Fatal("validation did not preserve the cross-package cycle")
	}
}

func TestValidateSharingRules(t *testing.T) {
	input := ValidationSharing{
		Numbers:  make(chan int),
		Callback: func() int { return 7 },
	}
	output, err := ValidateValidationSharing(input)
	if err != nil {
		t.Fatal(err)
	}
	if output.Numbers != input.Numbers || output.Callback() != 7 {
		t.Fatal("validation did not share safe channel and function values")
	}

	if _, err := ValidateValidationModelChannel(ValidationModelChannel{}); err == nil {
		t.Fatal("validation accepted a channel that can carry a tgo model")
	}
	if _, err := ValidateValidationModelAliases(ValidationModelAliases{}); err == nil {
		t.Fatal("validation accepted a function alias that can return a tgo model")
	}
	if _, err := ValidateValidationModelChannelAlias(ValidationModelChannelAlias{}); err == nil {
		t.Fatal("validation accepted a channel alias that can transport a tgo model")
	}
	if _, err := ValidateValidationDynamicSharing(ValidationDynamicSharing{}); err == nil {
		t.Fatal("validation accepted an interface channel or function")
	}
}

func TestValidateRejectsOpaquePrivateModel(t *testing.T) {
	account := model.NewAccountPersonal(model.AccountPersonal{Name: "A"})
	input := ValidationOpaqueEnvelope{
		Value: validationOpaque{account: account},
	}
	if _, err := ValidateValidationOpaqueEnvelope(input); err == nil {
		t.Fatal("validation accepted an opaque private field with a tgo model")
	}
	if _, err := ValidateValidationOpaqueNumberEnvelope(
		ValidationOpaqueNumberEnvelope{},
	); err == nil {
		t.Fatal("validation accepted an opaque private field")
	}
}
