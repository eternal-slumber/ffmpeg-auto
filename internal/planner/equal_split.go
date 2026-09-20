package planner

import (
	"fmt"
	"math"

	"content-factory/internal/model"
)

func EqualSplit(duration float64, parts int) ([]model.ClipPlan, error) {
	if duration <= 0 || math.IsNaN(duration) || math.IsInf(duration, 0) {
		return nil, fmt.Errorf("duration must be a finite positive number, got %v", duration)
	}
	if parts <= 0 {
		return nil, fmt.Errorf("parts must be positive, got %d", parts)
	}

	clips := make([]model.ClipPlan, parts)
	for i := range clips {
		start := duration * float64(i) / float64(parts)
		end := duration * float64(i+1) / float64(parts)
		if i == parts-1 {
			end = duration
		}
		clips[i] = model.ClipPlan{
			Number:   i + 1,
			Start:    start,
			Duration: end - start,
		}
	}
	return clips, nil
}
