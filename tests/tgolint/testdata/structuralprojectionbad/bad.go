package structuralprojectionbad

import (
	"example.com/tgolint/model"
	"example.com/tgolint/structuralprojection"
)

func Use() {
	structuralprojection.Primitive[model.Event]()()
	structuralprojection.Invalid[model.Event, model.Event]()()
	structuralprojection.Factory[model.Event]{}.ReceiverPrimitive()()
	structuralprojection.Pick[int, model.Event]()()
	structuralprojection.PickReceiver[int, model.Event]()()
}
