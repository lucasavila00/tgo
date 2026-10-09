package format_test

import (
	"testing"

	"tgo/pkg/format"
)

func TestSourceBasic(t *testing.T) {
	t.Parallel()
	input := "package sample\nfunc add(left int,right int)int{return left+right}\n"
	want := "package sample\n\nfunc add(left int, right int) int {\n\treturn left + right\n}\n"
	got, err := format.Source("sample.tgo", []byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", got, want)
	}
	again, err := format.Source("sample.tgo", got)
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != string(got) {
		t.Fatalf("second pass changed output:\n%s", again)
	}
}

func TestSourceRejectsMalformedInput(t *testing.T) {
	t.Parallel()
	_, err := format.Source("bad.tgo", []byte("package sample\nfunc {"))
	if err == nil {
		t.Fatal("format accepted malformed input")
	}
}
