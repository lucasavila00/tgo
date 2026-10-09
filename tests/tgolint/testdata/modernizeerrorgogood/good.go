package modernizeerrorgogood

import "fmt"

type record struct{}

func load() (*record, error) { return nil, nil }

func goSource() (*record, error) {
	value, err := load()
	if err != nil {
		return nil, fmt.Errorf("load: %w", err)
	}
	return value, nil
}
