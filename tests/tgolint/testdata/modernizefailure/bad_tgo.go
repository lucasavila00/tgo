package modernizefailure

import "errors"

type record struct{}

func pointer(err error) (*record, error) {
	return nil, err
}

func integer(err error) (int, error) {
	return 0, err
}

func many(err error) (bool, int, string, *record, error) {
	return false, 0, "", nil, err
}

func values(err error) (record, [0]int, error) {
	return record{}, [0]int{}, err
}

func named(err error) (value int, returned error) {
	return 0, err
}

var literal = func(err error) (string, error) {
	return "", err
}

func direct() (int, error) {
	return 0, errors.New("failure")
}
