package nilgood

import "example.com/tgolint/nilmodel"

func direct() {
	nilmodel.Need(&nilmodel.Item{})
}

func nestedPointers() {
	var maybeNil *nilmodel.Item
	outer := &maybeNil
	nilmodel.NeedOuter(outer)

	item := &nilmodel.Item{}
	inner := item
	both := &inner
	nilmodel.NeedInner(both)
	nilmodel.NeedBoth(both)
}

func narrow(value *nilmodel.Item) {
	if value == nil {
		return
	}
	nilmodel.Need(value)
}

func namedContract() {
	holder := nilmodel.NewHolder(&nilmodel.Item{})
	nilmodel.NeedHolder(holder)
}

func nilOuterPointer() {
	var holder *nilmodel.Holder = nil
	_ = holder
}

func guardAlias(value *nilmodel.Item) {
	valid := value != nil
	copyOfValid := valid
	if !copyOfValid {
		return
	}
	nilmodel.Need(value)
}

func valueAlias(value *nilmodel.Item) {
	candidate := value
	if candidate == nil {
		return
	}
	nilmodel.Need(value)
}

func shortCircuit(value *nilmodel.Item, enabled bool) {
	if value != nil && enabled {
		nilmodel.Need(value)
	}
	if value == nil || !enabled {
		return
	}
	nilmodel.Need(value)
}

func nilSwitch(value *nilmodel.Item) {
	switch value {
	case nil:
		return
	default:
		nilmodel.Need(value)
	}
}

func panicExit(value *nilmodel.Item) {
	if value == nil {
		panic("missing item")
	}
	nilmodel.Need(value)
}

func loop(value *nilmodel.Item) {
	for value == nil {
		value = &nilmodel.Item{}
	}
	nilmodel.Need(value)
}

func mapRead(values nilmodel.ItemMap, key string) {
	value, ok := values[key]
	present := ok
	if !present {
		return
	}
	nilmodel.Need(value)
}

func mapReadWithNilCheck(values nilmodel.ItemMap, key string) {
	value := values[key]
	if value == nil {
		return
	}
	nilmodel.Need(value)
}

func mapWrite(values nilmodel.ItemMap, key string, value *nilmodel.Item) {
	if value == nil {
		return
	}
	values[key] = value
	nilmodel.Need(values[key])
}

func channelRead(values nilmodel.ItemChan) {
	value, ok := <-values
	if !ok {
		return
	}
	nilmodel.Need(value)
}

func rangeValues(
	slice nilmodel.ItemSlice,
	mapping nilmodel.ItemMap,
	channel nilmodel.ItemChan,
) {
	for _, item := range slice {
		nilmodel.Need(item)
	}
	for _, item := range mapping {
		nilmodel.Need(item)
	}
	for item := range channel {
		nilmodel.Need(item)
	}
}

func assertion(value any) {
	item, ok := value.(*nilmodel.Item)
	if !ok || item == nil {
		return
	}
	nilmodel.Need(item)
}

func literals(item *nilmodel.Item) {
	if item == nil {
		return
	}
	_ = nilmodel.Holder{Required: item}
	_ = nilmodel.ItemArray{item, item}
	_ = nilmodel.ItemSlice{item}
	_ = nilmodel.ItemMap{"item": item}
}

func collectionOperations(item *nilmodel.Item) {
	if item == nil {
		return
	}
	values := make(nilmodel.ItemSlice, 0)
	values = append(values, item)
	copy(values, nilmodel.ItemSlice{item})
	_ = values
}

var handler nilmodel.RequiredHandler = nilmodel.AcceptOptional
