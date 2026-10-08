//tgolint:ignore-file

package suppressfile

import "example.com/tgolint/model"

func suppressedFile() {
	_ = model.Count{}
}
