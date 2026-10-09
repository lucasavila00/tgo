package nilbad

import "example.com/tgolint/nilmodel"

func argument(value *nilmodel.Item) {
	nilmodel.Need(value)
}

func nestedPointers(outer **nilmodel.Item) {
	nilmodel.NeedOuter(outer)
	nilmodel.NeedInner(outer)
	nilmodel.NeedBoth(outer)
}

func lostAtJoin(value *nilmodel.Item, condition bool) {
	if condition {
		if value == nil {
			return
		}
	}
	nilmodel.Need(value)
}

func badGuard(value *nilmodel.Item) {
	valid := value != nil
	value = nil
	if valid {
		nilmodel.Need(value)
	}
}

func mapWithoutProof(values nilmodel.ItemMap, key string) {
	value := values[key]
	nilmodel.Need(value)
}

func mapWrongBranch(values nilmodel.ItemMap, key string) {
	value, ok := values[key]
	if !ok {
		nilmodel.Need(value)
	}
}

func mapDelete(values nilmodel.ItemMap, key string, value *nilmodel.Item) {
	if value == nil {
		return
	}
	values[key] = value
	delete(values, key)
	nilmodel.Need(values[key])
}

func changedKey(values nilmodel.ItemMap, key string, value *nilmodel.Item) {
	if value == nil {
		return
	}
	values[key] = value
	key = "other"
	nilmodel.Need(values[key])
}

func channelWithoutProof(values nilmodel.ItemChan) {
	value := <-values
	nilmodel.Need(value)
}

func assertionOnlyProvesType(value any) {
	item, ok := value.(*nilmodel.Item)
	if !ok {
		return
	}
	nilmodel.Need(item)
}

func badLiterals(value *nilmodel.Item) {
	_ = nilmodel.Holder{}
	_ = nilmodel.Holder{Required: value}
	_ = nilmodel.ItemArray{&nilmodel.Item{}}
	_ = nilmodel.ItemSlice{value}
	_ = nilmodel.ItemMap{"item": value}
}

func badMakeAndClear() {
	values := make(nilmodel.ItemSlice, 2)
	clear(values)
}

func closureWrite(value *nilmodel.Item) {
	if value == nil {
		return
	}
	change := func() { value = nil }
	change()
	nilmodel.Need(value)
}

var badHandler nilmodel.OptionalHandler = nilmodel.RequireAndReturn

func badCrossPackageMethodContract(
	lookup *nilmodel.Lookup,
	value *nilmodel.Item,
) {
	_ = lookup.Find(value)
}
