package planner

import (
	"math"
	"testing"
)

func TestEqualSplit(t *testing.T) {
	const duration = 34.17
	clips, err := EqualSplit(duration, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(clips) != 10 {
		t.Fatalf("got %d clips, want 10", len(clips))
	}
	for i, clip := range clips {
		if clip.Number != i+1 {
			t.Fatalf("clip %d has number %d", i, clip.Number)
		}
		if i > 0 {
			previousEnd := clips[i-1].Start + clips[i-1].Duration
			if math.Abs(clip.Start-previousEnd) > 1e-12 {
				t.Fatalf("gap or overlap before clip %d", clip.Number)
			}
		}
	}
	last := clips[len(clips)-1]
	if math.Abs(last.Start+last.Duration-duration) > 1e-12 {
		t.Fatalf("plan ends at %v, want %v", last.Start+last.Duration, duration)
	}
}

func TestEqualSplitRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		duration float64
		parts    int
	}{
		{0, 10},
		{-1, 10},
		{math.NaN(), 10},
		{math.Inf(1), 10},
		{10, 0},
		{10, -1},
	}

	for _, test := range tests {
		if _, err := EqualSplit(test.duration, test.parts); err == nil {
			t.Fatalf("EqualSplit(%v, %d) returned no error", test.duration, test.parts)
		}
	}
}
