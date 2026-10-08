package suppressline

import "example.com/tgolint/model"

func suppressedLines() {
	//tgolint:ignore
	_ = model.Count{}
	_ = model.Count{} //tgolint:ignore
}
