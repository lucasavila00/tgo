package navigation

import "testing"

func TestPathURIValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		path   string
		volume string
		want   string
	}{
		{
			name: "Unix",
			path: "/work space/Café#1.tgo",
			want: "file:///work%20space/Caf%C3%A9%231.tgo",
		},
		{
			name: "Windows drive",
			path: "C:/work space/main.tgo", volume: "C:",
			want: "file:///C:/work%20space/main.tgo",
		},
		{
			name: "Windows share",
			path: "//server/share/work space/main.tgo", volume: "//server/share",
			want: "file://server/share/work%20space/main.tgo",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := pathURIValue(test.path, test.volume); got != test.want {
				t.Fatalf("URI = %q, want %q", got, test.want)
			}
		})
	}
}
