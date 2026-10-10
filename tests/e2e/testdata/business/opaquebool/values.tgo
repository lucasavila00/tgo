package opaquebool

import "errors"

type privateBool bool

var ErrCheck = errors.New("opaque bool check failure")

func Factory() privateBool { return true }

func Check(fail bool) (privateBool, error) {
	if fail {
		return false, ErrCheck
	}
	return true, nil
}

func ConsumeAny(value any, later int) bool {
	_, ok := value.(privateBool)
	return ok && later == 7
}
