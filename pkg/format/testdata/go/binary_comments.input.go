package gaps

func flags(op1, op2 int) int {
	return (0xe << 24) | // opcode
		(op1 << 20) | // first operand
		(op2 << 16) /* second operand */ |
		1
}
