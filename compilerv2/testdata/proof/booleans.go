package proof

type Flag bool

func CaseBooleans(err error) error {
	observe(booleanEffect("left", Reached) && booleanEffect("right", true))
	var named Flag = Flag(booleanEffect("named", true))
	observe(bool(named))
	return nil
}
