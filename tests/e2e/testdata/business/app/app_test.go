package app_test

import (
	"encoding/json"
	"errors"
	"testing"
	"unsafe"

	"example.com/business/app"
	"example.com/business/legacy"
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
	firstNotice := app.Notice("one")
	secondNotice := app.Notice("two")
	firstLabels := model.NoticeLabels(firstNotice)
	firstLabels["shared"] = "yes"
	if model.NoticeLabels(firstNotice)["shared"] != "yes" ||
		len(model.NoticeLabels(secondNotice)) != 0 {
		t.Fatal("variant defaults lost freshness or aliases")
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

func TestGoInterop(t *testing.T) {
	person := model.Personal("Lucas")
	store := &legacy.MemoryStore{
		Accounts: map[legacy.AccountID]model.Account{"one": person},
	}
	label, err := app.StoredLabel(store, "one")
	if err != nil || label != "Lucas" {
		t.Fatalf("stored label: %q, %v", label, err)
	}
	_, err = app.StoredLabel(store, "missing")
	if !errors.Is(err, legacy.ErrMissing) {
		t.Fatalf("error identity: %v", err)
	}
	labels, err := app.Labels([]model.Account{person, model.Personal("Other")})
	if err != nil || labels[0] != "Lucas" || labels[1] != "Other" {
		t.Fatalf("generic callback: %v, %v", labels, err)
	}
	if app.FirstStreamLabel(person) != "Lucas" {
		t.Fatal("channel or variadic call")
	}
	app.Replace(&person, "Changed")
	if model.Label(person) != "Changed" {
		t.Fatal("pointer or callback identity")
	}
	if app.ForeignTypedNil() == nil {
		t.Fatal("typed nil error identity was lost")
	}
	if app.ForeignVariadic("values", "a", "b") != "values:[a b]" {
		t.Fatal("variadic values")
	}
}

func TestGeneratedCost(t *testing.T) {
	type accountLayout struct {
		tag      uint8
		personal model.AccountPersonal
		business model.AccountBusiness
	}
	if unsafe.Sizeof(model.Account{}) != unsafe.Sizeof(accountLayout{}) {
		t.Fatal("enum layout differs from the direct Go layout")
	}
	person := model.Personal("Lucas")
	var label string
	allocations := testing.AllocsPerRun(1000, func() {
		label = model.Label(person)
	})
	if allocations != 0 || label != "Lucas" {
		t.Fatalf("match cost: %f allocations, %q", allocations, label)
	}
}
