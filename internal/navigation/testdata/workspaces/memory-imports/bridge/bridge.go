package bridge

import "time"

func Normalize(value time.Time) time.Time {
	return value.UTC()
}
