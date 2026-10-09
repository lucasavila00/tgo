package effectfactbad

import (
	"example.com/tgolint/effectfactsource"
	"example.com/tgolint/model"
)

func Invalid() {
	effectfactsource.ConditionalZero[model.Event](true)
}
