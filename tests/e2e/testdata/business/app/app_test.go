package app_test

import (
	"encoding/json"
	"testing"

	"example.com/business/app"
	"example.com/business/model"
)

func TestBusiness(t *testing.T) {
	if got := app.Summary("Lucas"); got != "person Lucas" {
		t.Fatal(got)
	}
	quantity, err := model.NewQuantity(3)
	if err != nil || quantity.Value() != 3 {
		t.Fatalf("quantity: %v, %v", quantity, err)
	}
	if _, err := model.NewQuantity(0); err == nil {
		t.Fatal("zero passed the constructor")
	}
	first := app.Request("a")
	second := model.NewRequest("b")
	copy := first
	copy.Tags["shared"] = "yes"
	if first.Tags["shared"] != "yes" || len(second.Tags) != 0 {
		t.Fatal("defaults lost freshness or aliases")
	}
	account := model.Business("Acme", []model.Account{model.Personal("Lucas")}, first.Tags)
	if model.Label(account) != "Acme" {
		t.Fatal("wrong variant")
	}
	// Go can violate the contract. Value must not add a runtime check.
	bad := model.Quantity{}
	if bad.Value() != 0 {
		t.Fatal("foreign zero did not pass through")
	}
	sorted := app.Sorted([]int{3, 1, 2})
	if sorted[0] != 1 || sorted[2] != 3 {
		t.Fatal(sorted)
	}
	message := model.NewMessage("one")
	encoded, err := json.Marshal(message)
	if err != nil || string(encoded) != `{"id":"one","tags":{}}` {
		t.Fatalf("message: %s, %v", encoded, err)
	}
}
