package parity

type alignmentRow struct {
	in      string
	want    string
	wantErr error
}

var alignmentRows = []alignmentRow{
	{
		in:   "input",
		want: "output",
		wantErr: makeError("failure").
			withDetails("context"),
	},
	{
		in:      "next input",
		want:    "next output",
		wantErr: makeError("next failure"),
	},
}
