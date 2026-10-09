package effectfactsource

func ConditionalZero[T any](enabled bool) {
	if enabled {
		var value T
		_ = value
	}
}
