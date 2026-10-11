package foreign

type private int

var Number = 0

func Value(value int) private { return private(value) }
func Use(value private)       {}
