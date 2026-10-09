package app_test

import (
	"testing"

	"example.com/business/app"
	"example.com/business/model"
)

func TestCollections(t *testing.T) {
	person := model.Personal("Lucas")
	company := model.Business("Acme", []model.Account{person}, nil)
	accounts := map[string]model.Account{"present": person}
	if app.Lookup(accounts, "present") != "Lucas" || app.Lookup(accounts, "absent") != "missing" {
		t.Fatal("map presence")
	}
	channel := make(chan model.Account, 1)
	channel <- company
	close(channel)
	if app.Receive(channel) != "Acme" || app.Receive(channel) != "closed" {
		t.Fatal("channel presence")
	}
	if app.Assert(person) != "Lucas" || app.Assert("wrong type") != "other" {
		t.Fatal("assertion presence")
	}
	numbers := make(chan int)
	close(numbers)
	if value, ok := app.ReadInt(numbers); value != 0 || ok {
		t.Fatal("zero-valid channel read")
	}
	if app.Nested(company) != "Lucas" {
		t.Fatal("nested tag switch")
	}
	values := []model.Account{person, company, model.Personal("Other")}
	if len(app.Shorten(values, 2)) != 2 {
		t.Fatal("bounded reslice")
	}
	app.Overlap(values)
	if model.Label(values[1]) != "Lucas" || model.Label(values[2]) != "Acme" {
		t.Fatal("overlapping copy")
	}
	app.ClearMap(accounts)
	if len(accounts) != 0 {
		t.Fatal("clear map")
	}
}
