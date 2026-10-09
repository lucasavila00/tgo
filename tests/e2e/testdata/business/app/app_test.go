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
	point, err := model.NewPositivePoint(struct{ X int }{X: 1})
	if err != nil || point.Value().X != 1 {
		t.Fatalf("checked struct: %v, %v", point, err)
	}
	multiline, err := model.NewMultiline(1)
	if err != nil || multiline.Value() != 1 {
		t.Fatalf("multiline checked type: %v, %v", multiline, err)
	}
	whereValue, enumValue := model.ContextualTypeNames()
	if whereValue != 1 || enumValue != 2 {
		t.Fatal("contextual type names became tgo keywords")
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
	if model.AliasLabel(account) != "Acme" {
		t.Fatal("alias lost the model rules")
	}
	boxed, ok := model.AsAny(account).(model.Account)
	if !ok || model.Label(boxed) != "Acme" {
		t.Fatal("interface conversion changed the model value")
	}
	if model.CounterValue(model.Counter(2)) != 2 {
		t.Fatal("named struct matched a private model layout")
	}
	if model.AnonymousPoint().X != 1 {
		t.Fatal("ordinary struct reserved a model layout")
	}
	if model.Tag(model.Tagged{}) != 0 {
		t.Fatal("ordinary tgoTag field became private")
	}
	person := model.Personal("Lucas")
	if model.OrdinaryMatchName() != "ordinary" {
		t.Fatal("match is not an ordinary identifier")
	}
	model.OrdinaryMatchStatement()
	if model.MatchLabel(0) != 1 {
		t.Fatal("match is not an ordinary label")
	}
	if model.FunctionTagSubject(person) != "Lucas" {
		t.Fatal("function tag subject used the wrong body")
	}
	if model.LiteralTagSubject("Literal") != "Literal" {
		t.Fatal("literal tag subject used the wrong body")
	}
	if model.SignalName(model.NewSignalOn(model.SignalOn{})) != "on" {
		t.Fatal("multiline enum declaration has the wrong tag")
	}
	if model.SignalState(model.NewSignalOff(model.SignalOff{})) != "known" {
		t.Fatal("multi-tag case rejected a known tag")
	}
	if model.SignalStateOrInvalid(model.NewSignalOn(model.SignalOn{})) != "known" {
		t.Fatal("returning default rejected a known tag")
	}
	explicitFlag, explicitName := model.MarkerValues(model.ExplicitMarker())
	defaultFlag, defaultName := model.MarkerValues(model.SelectedMarker())
	if !explicitFlag || explicitName != "set" || defaultFlag || defaultName != "default" {
		t.Fatal("default marker captured a source field")
	}
	if model.Label(model.AliasAccount("Alias")) != "Alias" ||
		model.Label(app.ImportedAlias("Imported")) != "Imported" ||
		model.Label(app.LocalImportedAlias("Local")) != "Local" {
		t.Fatal("alias variant construction failed")
	}
	if request := app.LocalImportedRequest("local"); request.ID != "local" || len(request.Tags) != 0 {
		t.Fatal("alias default construction failed")
	}
	if model.LabeledTagSwitch(person) != "done" {
		t.Fatal("label did not label the tag switch")
	}
	reexported, ok := app.ReexportedAccount("Bridge").(model.Account)
	if !ok || model.Label(reexported) != "Bridge" {
		t.Fatal("re-exported enum alias used the wrong owner")
	}
	onlyReexported, ok := app.OnlyReexportedAccount("Only").(model.Account)
	if !ok || model.Label(onlyReexported) != "Only" {
		t.Fatal("lowered enum alias lost its bridge import")
	}
	goReexported, ok := app.GoReexportedAccount("Go bridge").(model.Account)
	if !ok || model.Label(goReexported) != "Go bridge" {
		t.Fatal("Go enum alias used the wrong owner")
	}
	goRequest, ok := app.GoReexportedRequest("Go request").(model.Request)
	if !ok || goRequest.ID != "Go request" || len(goRequest.Tags) != 0 {
		t.Fatal("Go default alias used the wrong owner")
	}
	reexportedRequest, ok := app.ReexportedRequest("bridge").(model.Request)
	if !ok || reexportedRequest.ID != "bridge" || len(reexportedRequest.Tags) != 0 {
		t.Fatal("re-exported default alias used the wrong owner")
	}
	completeRequest, ok := app.CompleteReexportedRequest("complete").(model.Request)
	if !ok || completeRequest.ID != "complete" || len(completeRequest.Tags) != 0 {
		t.Fatal("complete default selection added an unused owner")
	}
	exposed := model.ExposedQuantity{}
	if model.ConvertExposed[model.ExposedQuantity](exposed) != exposed {
		t.Fatal("constraint intersection admitted a checked type")
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
	if unsafe.Sizeof(model.Signal{}) != unsafe.Sizeof(uint8(0)) {
		t.Fatal("payload-free enum is larger than its tag")
	}
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
		t.Fatalf("tag switch cost: %f allocations, %q", allocations, label)
	}
}
