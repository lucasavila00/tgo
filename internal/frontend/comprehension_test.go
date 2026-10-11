package frontend

import (
	"go/token"
	"testing"
)

func TestParseComprehensions(t *testing.T) {
	for _, expression := range []string{
		`[]int{x for _, x := range xs}`,
		"[]string{`a\r\nb` for range 1}",
		"[]int{x /* output\r\ncomment */ for _,x := range xs /* source\r\ncomment */}",
		`map[int]string{x: f(x) for _, x := range xs if x > 0}`,
		`Named[int]{x+y for _, x := range xs for _, y := range ys}`,
		`[]int{x for _, x := range []int{y for _, y := range ys}}`,
		`[]int{if x > 0 {x} else {0} for _, x := range xs}`,
		`[]int{x for _, x := range if true {xs} else {ys} if x > 0}`,
		"[]int{\n x\n for _, x := range xs\n if x > 0,\n}",
		"[]int{x // output\n for _, x := range xs // source\n if x > 0 // filter\n}",
	} {
		t.Run(expression, func(t *testing.T) {
			parsed, err := ParseFile(token.NewFileSet(), "main_tgo.go", "main.tgo", []byte("package p\nvar result = "+expression), make(map[string]bool))
			if err != nil {
				t.Fatal(err)
			}
			if len(parsed.Comprehensions) == 0 {
				t.Fatal("missing comprehension")
			}
			for _, record := range parsed.Comprehensions {
				if record.Type == nil || record.Body == nil || record.Position.Filename != "main.tgo" {
					t.Fatalf("incomplete record: %+v", record)
				}
			}
		})
	}
}

func TestComprehensionsKeepOrdinaryLoops(t *testing.T) {
	input := []byte(`package p; func g(xs []int) []int { r := []int{}; for _,x := range xs { r=append(r,x) }; return r }; func f() { for range 2 { print(1); for range 3 {} }; _ = []int{x for x := range 3} }`)
	parsed, err := ParseFile(token.NewFileSet(), "main_tgo.go", "main.tgo", input, make(map[string]bool))
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Comprehensions) != 1 {
		t.Fatalf("got %d comprehensions", len(parsed.Comprehensions))
	}
}
