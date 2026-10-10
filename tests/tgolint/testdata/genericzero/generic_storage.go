package genericzero

type FunctionBox struct {
	Values []func()
}

func storedEffect[T any]() {
	var value T
	_ = value
}

func ReturnedAppendedAlias[T any]() []func() {
	values := []func(){storedEffect[T]}
	return append(values)
}

func CopyAlias[T any](target []func()) {
	values := []func(){storedEffect[T]}
	copy(target, values)
}

func LocalLiteralContainer[T any]() {
	values := []func(){func() {
		var value T
		_ = value
	}}
	values[0]()
}

func LocalAssignedContainer[T any]() {
	values := make([]func(), 1)
	values[0] = func() {
		var value T
		_ = value
	}
	values[0]()
}

func ReturnedContainer[T any]() []func() {
	values := []func(){func() {
		var value T
		_ = value
	}}
	return values
}

func ReturnedAnyContainer[T any]() []any {
	values := []any{func() {
		var value T
		_ = value
	}}
	return values
}

func EscapedAnyContainer[T any]() {
	values := []any{func() {
		var value T
		_ = value
	}}
	keepValue(values)
}

func ForwardedLocalCall[T any]() {
	value := forwardNested(func() {
		var item T
		_ = item
	})
	value()
}

func ForwardedUnknownEscape[T any]() {
	keepValue(forwardNested(func() {
		var value T
		_ = value
	}))
}

func maybeForward(enabled bool, value func()) func() {
	if enabled {
		value()
	}
	return value
}

func neverForward(value func()) func() {
	if false {
		value()
	}
	return value
}

func ConditionalForwardCall[T any](enabled bool) {
	_ = maybeForward(enabled, func() {
		var value T
		_ = value
	})
}

func NeverForwardCall[T any]() {
	_ = neverForward(func() {
		var value T
		_ = value
	})
}

func callThenSafe(value func()) func() {
	(func() {
		value()
	})()
	return func() {}
}

func discardCapture(value func()) func() {
	_ = func() {
		value()
	}
	return func() {}
}

func DirectInvokedForwardCall[T any]() {
	_ = callThenSafe(func() {
		var value T
		_ = value
	})
}

func DiscardedForwardCapture[T any]() {
	_ = discardCapture(func() {
		var value T
		_ = value
	})
}

func captureButDoNotUse(value func()) func() {
	unused := func() {
		value()
	}
	_ = unused
	return func() {}
}

func NestedUnusedForwardCapture[T any]() {
	_ = captureButDoNotUse(func() {
		var value T
		_ = value
	})
}

func DeadAlias[T any]() func() {
	value := func() {}
	if false {
		value = func() {
			var item T
			_ = item
		}
	}
	return value
}

func ConditionalAlias[T any](enabled bool) func() {
	value := func() {}
	if enabled {
		value = func() {
			var item T
			_ = item
		}
	}
	return value
}

func SafeContainerLength[T any]() int {
	values := []func(){func() {
		var value T
		_ = value
	}}
	return len(values)
}

func OverwrittenContainer[T any]() {
	values := []func(){func() {
		var value T
		_ = value
	}}
	values[0] = func() {}
	values[0]()
}

func UnusedCapturedClosure[T any]() {
	value := func() {
		var item T
		_ = item
	}
	_ = value
}

func ReturnAssignedBox[T any]() FunctionBox {
	var box FunctionBox
	box.Values = []func(){func() {
		var item T
		_ = item
	}}
	return box
}
