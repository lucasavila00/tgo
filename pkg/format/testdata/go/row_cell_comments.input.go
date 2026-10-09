package alignment

var selectorKeys = map[string]bool{
	context.GOOS:true,
	context.GOARCH:false,
}

var binaryKeys = map[int]string{
	baseChunkIndex:"base",
	baseChunkIndex+1:"one",
	baseChunkIndex+0xe:"last",
}

func statementComments(buffer readBuffer) {
	buffer.uint32() // reserved
	for len(buffer)>=4 { // need at least four bytes
		buffer.uint16()
	}
	if len(buffer)>0 {
		buffer.uint8()
	} // consumed the remainder
	buffer.uint64() // final value
}

var traceStart = lineNumber() // start
func traceFirst(value int) { // first
	traceValues[value]++ // body
} // end
func traceSecond(value int) { // second
	traceValues[value]++ // body
} // end
var traceValues [2]int // values

func Environments() []string{return environments}
func SetEnvironments(values []string){environments=values}
