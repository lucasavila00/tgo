package transpiler_test

import (
	"reflect"
	"testing"

	"github.com/lucasavila00/tgo/pkg/transpiler"
)

func TestSourcePublicAPI(t *testing.T) {
	sourceType := reflect.TypeOf((*transpiler.Source)(nil))
	if sourceType.NumMethod() != 1 || sourceType.Method(0).Name != "InsertAfter" {
		t.Fatalf("Source exposes unsupported methods: %v", sourceType)
	}
	var source transpiler.Source
	if _, err := source.InsertAfter(transpiler.Site{}, nil); err == nil ||
		err.Error() != "expression site does not belong to this source" {
		t.Fatalf("invalid site: %v", err)
	}
}
