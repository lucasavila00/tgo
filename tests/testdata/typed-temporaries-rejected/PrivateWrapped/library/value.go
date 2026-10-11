package library

type private int

func Value() private                  { return 7 }
func Use(value private, next int) int { return int(value) + next }
