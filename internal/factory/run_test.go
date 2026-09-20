package factory

import "testing"

func TestNumberedOutputPath(t *testing.T) {
	tests := []struct {
		first  string
		number int
		want   string
	}{
		{"storage/output/clip-001.mp4", 1, "storage/output/clip-001.mp4"},
		{"storage/output/clip-001.mp4", 2, "storage/output/clip-002.mp4"},
		{"/tmp/result.mp4", 3, "/tmp/result-003.mp4"},
	}
	for _, test := range tests {
		if got := numberedOutputPath(test.first, test.number); got != test.want {
			t.Fatalf("numberedOutputPath(%q, %d) = %q, want %q", test.first, test.number, got, test.want)
		}
	}
}
