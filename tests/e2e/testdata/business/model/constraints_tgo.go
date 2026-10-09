package model

type ExposedQuantity struct {
	value int
}

type OnlyExposedQuantity interface {
	Quantity | ExposedQuantity
	ExposedQuantity
}

func ConvertExposed[T OnlyExposedQuantity](value ExposedQuantity) T {
	return T(value)
}
