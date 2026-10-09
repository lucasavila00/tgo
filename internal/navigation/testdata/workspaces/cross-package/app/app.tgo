package app

import "example.test/navigation/lib"

func Read() string {
	return lib.Target(lib.Record{Name: "app"})
}

func Scope() string {
	value := "outer"
	{
		value := "inner"
		_ = value
	}
	return value
}
