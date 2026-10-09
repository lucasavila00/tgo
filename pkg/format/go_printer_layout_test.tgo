package format

import (
	"testing"
	"text/tabwriter"
)

func TestFinishGoPrinterOutput(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "alignment",
			raw:  "\tshort\v// one\n\tlonger\v// two\f",
			want: "\tshort  // one\n\tlonger // two\n",
		},
		{
			name: "section",
			raw:  "\ta\v// one\f\tlonger\v// two\n",
			want: "\ta // one\n\tlonger // two\n",
		},
		{
			name: "unicode",
			raw:  "\tŝ\v// one\n\tlong\v// two\n",
			want: "\tŝ    // one\n\tlong // two\n",
		},
		{
			name: "escaped text",
			raw: string([]byte{
				tabwriter.Escape,
				'/', '/', 'x', '\v', 'y',
				tabwriter.Escape,
				' ', ' ', '\n',
			}),
			want: "//x\vy\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := finishGoPrinterOutput([]byte(test.raw))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != test.want {
				t.Fatalf("output = %q, want %q", got, test.want)
			}
		})
	}
}
