package foreign

type private int

func Value(value int) private { return private(value) }
func Use(value private)       {}
