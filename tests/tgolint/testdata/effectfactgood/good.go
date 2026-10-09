package effectfactgood

import (
	"example.com/tgolint/effectfactsource"
	"example.com/tgolint/model"
)

func Valid() {
	effectfactsource.ConditionalZero[model.Event](false)
}
