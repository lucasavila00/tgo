//go:build loweringfuzz

package tests

import (
	"bytes"
	"fmt"
	"go/format"
	"go/scanner"
	"go/token"
	"runtime"
	"strings"
	"testing"
)

// Paired expressions share leaf calls, but the Go reference uses functions
// for value-only branches. Source returns, defers, and jumps use separate templates.
type ifFuzzExpression struct{ source, reference string }
type ifFuzzInput struct {
	data          []byte
	offset, calls int
}

func (input *ifFuzzInput) next() byte {
	value := input.data[input.offset%len(input.data)]
	input.offset++
	return value
}

func (input *ifFuzzInput) expression(depth int) ifFuzzExpression {
	if depth == 0 {
		id := input.calls
		input.calls++
		call := fmt.Sprintf("read(%d, %d)", id, input.next()%8)
		return ifFuzzExpression{call, call}
	}
	switch input.next() % 4 {
	case 0:
		left, right := input.expression(depth-1), input.expression(depth-1)
		return ifFuzzExpression{"(" + left.source + " + " + right.source + ")", "(" + left.reference + " + " + right.reference + ")"}
	case 1:
		left, right := input.expression(depth-1), input.expression(depth-1)
		return ifFuzzExpression{"add(" + left.source + ", " + right.source + ")", "add(" + left.reference + ", " + right.reference + ")"}
	default:
		id := input.calls
		input.calls++
		flag := input.next()&1 != 0
		yes, no := input.expression(depth-1), input.expression(depth-1)
		condition := fmt.Sprintf("test(%d, %t)", id, flag)
		source := fmt.Sprintf("(if %s { mark(%d); value := %s; value } else { mark(%d); %s })", condition, id, yes.source, id+100, no.source)
		reference := fmt.Sprintf("func() int { if %s { mark(%d); value := %s; return value }; mark(%d); return %s }()", condition, id, yes.reference, id+100, no.reference)
		return ifFuzzExpression{source, reference}
	}
}

var ifFuzzBodies = []struct{ source, reference string }{
	{`return EXPR`, `return EXPR`},
	{`return add(read(80, 1), EXPR)`, `return add(read(80, 1), EXPR)`},
	{`values := []int{read(80, 1), EXPR, read(81, 2)}; return values[0]+values[1]+values[2]`, `values := []int{read(80, 1), EXPR, read(81, 2)}; return values[0]+values[1]+values[2]`},
	{`values := [2]int{}; values[if FLAG {mark(80); 0} else {mark(81); 1}] = EXPR; return values[0]+values[1]`, `values := [2]int{}; index := 0; if FLAG {mark(80)} else {mark(81); index=1}; values[index] = EXPR; return values[0]+values[1]`},
	{`return if FLAG {defer mark(80); EXPR} else {defer mark(81); EXPR}`, `if FLAG {defer mark(80); return EXPR}; defer mark(81); return EXPR`},
	{`value := if FLAG {return EXPR} else {EXPR}; return value+1`, `if FLAG {return EXPR}; value := EXPR; return value+1`},
	{`var value int = if FLAG {return EXPR} else {return EXPR}; return value`, `if FLAG {return EXPR}; return EXPR`},
	{`value := if FLAG {goto done} else {EXPR}; mark(value); done: return 7`, `if FLAG {goto done}; {value := EXPR; mark(value)}; done: return 7`},
	{`sum := 0; for i:=0; i<3; i++ {value := if i==1 {continue} else {EXPR}; sum+=value}; return sum`, `sum := 0; for i:=0; i<3; i++ {if i==1 {continue}; value := EXPR; sum+=value}; return sum`},
	{`sum := 0; outer: for i:=0; i<3; i++ {value := if i==1 {break outer} else {EXPR}; sum+=value}; return sum`, `sum := 0; outer: for i:=0; i<3; i++ {if i==1 {break outer}; value := EXPR; sum+=value}; return sum`},
	{`value := if FLAG {step()OP; EXPR} else {EXPR}; return value`, `var value int; if FLAG {if _, e:=step(); e!=nil {failureReturn(e)}; value=EXPR} else {value=EXPR}; return value`},
	{`var small int8 = if FLAG {127} else {0}; return int(small+int8(1))+EXPR`, `var small int8; if FLAG {small=127}; return int(small+int8(1))+EXPR`},
	{`switch EXPR {case 0: return 1; default: return 2}`, `switch EXPR {case 0: return 1; default: return 2}`},
	{`channel := make(chan int,1); channel<-5; select {case value:= <-(if FLAG {mark(80); channel} else {mark(81); channel}): return value+EXPR}`, `channel := make(chan int,1); channel<-5; if FLAG {mark(80)} else {mark(81)}; select {case value:= <-channel: return value+EXPR}`},
	{`value := 0; if false && (if FLAG {mark(80); true} else {mark(81); false}) {value=1}; return value+EXPR`, `value := 0; return value+EXPR`},
	{`value := 0; for range if FLAG {mark(80); [2]int{}} else {mark(81); [2]int{}} {value++}; return value+EXPR`, `value := 0; if FLAG {mark(80)} else {mark(81)}; for range [2]int{} {value++}; return value+EXPR`},
}

func ifFuzzPrograms(t *testing.T, data []byte) (string, string) {
	t.Helper()
	input := &ifFuzzInput{data: data, offset: 4}
	expression := input.expression(1 + int(data[1]%3))
	body := ifFuzzBodies[int(data[0])%len(ifFuzzBodies)]
	flag, fail, wrapped := data[2]&1 != 0, data[3]&1 != 0, data[3]&2 != 0
	op, context := "!", ""
	if wrapped {
		op, context = "!!", "step: "
	}
	replace := func(body, expr string) string {
		return strings.NewReplacer("EXPR", expr, "FLAG", fmt.Sprint(flag), "OP", op).Replace(body)
	}
	common := fmt.Sprintf(`package main
import ("errors"; "fmt"; "strings")
var trace strings.Builder
var sentinel = errors.New("failure")
func mark(id int) {fmt.Fprintf(&trace,"m%%d;",id)}
func read(id, value int) int {fmt.Fprintf(&trace,"r%%d;",id); return value}
func test(id int, value bool) bool {fmt.Fprintf(&trace,"t%%d;",id); return value}
func add(a,b int) int {return a+b}
func step() (int,error) {mark(90); if %t {return 0,sentinel}; return 1,nil}
func main() {
 defer func(){if p:=recover();p!=nil {fmt.Printf("%%s|panic:%%T\n",trace.String(),p)}}()
 value,err:=run()
 fmt.Printf("%%s|value:%%d|error:%%v|same:%%t|is:%%t\n",trace.String(),value,err,err==sentinel,errors.Is(err,sentinel))
}
`, fail)
	// Both programs report the named result after source deferred calls.
	sourceBody := replace(body.source, expression.source)
	referenceBody := replace(body.reference, expression.reference)
	source := common + "func run() (result int, err error) {defer func(){mark(result)}(); " + ifFuzzReturns(sourceBody) + "}\n"
	reference := common + fmt.Sprintf(`
type stopped struct {err error}
func failureReturn(err error) {panic(stopped{fmt.Errorf(%q+"%%w",err)})}
func value() int { %s }
func run() (result int, err error) {
 defer func(){if p:=recover();p!=nil {if stop,ok:=p.(stopped);ok {err=stop.err} else {panic(p)}}}()
 defer func(){mark(result)}()
 result=value()
 return result,nil
}
`, context, referenceBody)
	if !wrapped {
		reference = strings.Replace(reference, `panic(stopped{fmt.Errorf(""+"%w",err)})`, `panic(stopped{err})`, 1)
	}
	formatted, err := format.Source([]byte(reference))
	if err != nil {
		t.Fatalf("invalid reference: %v\n%s", err, reference)
	}
	return source, string(formatted)
}

// ifFuzzReturns adds the source function's error result to template returns.
func ifFuzzReturns(body string) string {
	set := token.NewFileSet()
	file := set.AddFile("", -1, len(body))
	var scan scanner.Scanner
	scan.Init(file, []byte(body), nil, 0)
	var ends []int
	active, depth := false, 0
	for {
		position, kind, _ := scan.Scan()
		if kind == token.EOF {
			break
		}
		if !active {
			if kind == token.RETURN {
				active = true
				depth = 0
			}
			continue
		}
		if depth == 0 && (kind == token.SEMICOLON || kind == token.RBRACE) {
			ends = append(ends, file.Offset(position))
			active = false
			continue
		}
		switch kind {
		case token.LPAREN, token.LBRACK, token.LBRACE:
			depth++
		case token.RPAREN, token.RBRACK, token.RBRACE:
			depth--
		}
	}
	for i := len(ends) - 1; i >= 0; i-- {
		end := ends[i]
		body = body[:end] + ", nil" + body[end:]
	}
	return body
}

func FuzzIfExpressionBehavior(f *testing.F) {
	for i := range ifFuzzBodies {
		for _, flags := range []byte{0, 1, 2, 3} {
			f.Add([]byte{byte(i), 1, flags, flags, 2, 1, 0, 3, 4, 5})
		}
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 4 {
			return
		}
		source, reference := ifFuzzPrograms(t, data)
		module := []byte("module example.com/iffuzz\n\ngo " + strings.TrimPrefix(runtime.Version(), "go") + "\n")
		referenceDir := t.TempDir()
		writeFiles(t, referenceDir, map[string][]byte{"go.mod": module, "main.go": []byte(reference)})
		want := fuzzRun(t, referenceDir, reference)
		sourceDir := t.TempDir()
		writeFiles(t, sourceDir, map[string][]byte{"go.mod": module, "main.tgo": []byte(source)})
		files := fuzzCompile(t, sourceDir, source)
		output := t.TempDir()
		writeFiles(t, output, files)
		got := fuzzRun(t, output, source)
		if !bytes.Equal(got, want) {
			t.Fatalf("behavior differs\nINPUT:\n%s\nREFERENCE:\n%s\nWANT:\n%s\nGOT:\n%s\nGENERATED:\n%s", source, reference, want, got, files["main_tgo.go"])
		}
	})
}
