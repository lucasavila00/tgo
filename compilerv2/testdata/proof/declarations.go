package proof

func CaseDeclarations(err error) error {
	x := effect("first")
	var (
		y = effect("second")
		z = x + y
	)
	{
		x := effect("shadow")
		consume(x, z)
	}
	consume(x, y)
	return nil
}
