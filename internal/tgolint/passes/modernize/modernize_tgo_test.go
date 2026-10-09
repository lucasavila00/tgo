package modernize

import (
	"go/constant"
	"go/types"
	"testing"
)

func TestZeroValueClassification(t *testing.T) {
	if !zeroConstant(constant.MakeInt64(0)) {
		t.Fatal("zero integer was not classified as zero")
	}
	if zeroConstant(constant.MakeInt64(1)) {
		t.Fatal("nonzero integer was classified as zero")
	}
	if !predeclaredError(types.Universe.Lookup("error").Type()) {
		t.Fatal("predeclared error was not classified as error")
	}
}
