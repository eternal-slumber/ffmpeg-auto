package planner

import (
	"fmt"
	"math"
	"testing"
)

func TestPlanInterruptions(t *testing.T) {
	tests := []struct {
		clipDuration float64
		want         []float64
	}{
		{30, []float64{15}},
		{60, []float64{30}},
		{89, []float64{44.5}},
		{90, []float64{45}},
		{91, []float64{30, 60, 90}},
		{120, []float64{30, 60, 90}},
		{130, []float64{30, 60, 90, 120}},
		{180, []float64{30, 60, 90, 120, 150}},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%g_seconds", test.clipDuration), func(t *testing.T) {
			got, err := PlanInterruptions(test.clipDuration, 5.027)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(test.want) {
				t.Fatalf("got %d interruptions, want %d: %+v", len(got), len(test.want), got)
			}
			for i, want := range test.want {
				if got[i].At != want || got[i].Duration != 5.027 {
					t.Fatalf("interruption %d = %+v, want at=%v duration=5.027", i, got[i], want)
				}
			}
		})
	}
}

func TestPlanInterruptionsRejectsInvalidInput(t *testing.T) {
	tests := [][2]float64{
		{0, 5},
		{-1, 5},
		{math.NaN(), 5},
		{math.Inf(1), 5},
		{30, 0},
		{30, -1},
		{30, math.NaN()},
		{30, math.Inf(1)},
	}
	for _, test := range tests {
		if _, err := PlanInterruptions(test[0], test[1]); err == nil {
			t.Fatalf("PlanInterruptions(%v, %v) returned no error", test[0], test[1])
		}
	}
}
