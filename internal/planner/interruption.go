package planner

import (
	"fmt"
	"math"

	"content-factory/internal/model"
)

func PlanInterruptions(clipDuration, bannerDuration float64) ([]model.Interruption, error) {
	if clipDuration <= 0 || math.IsNaN(clipDuration) || math.IsInf(clipDuration, 0) {
		return nil, fmt.Errorf("clip duration must be a finite positive number, got %v", clipDuration)
	}
	if bannerDuration <= 0 || math.IsNaN(bannerDuration) || math.IsInf(bannerDuration, 0) {
		return nil, fmt.Errorf("banner duration must be a finite positive number, got %v", bannerDuration)
	}

	if clipDuration <= 90 {
		return []model.Interruption{{At: clipDuration / 2, Duration: bannerDuration}}, nil
	}

	var interruptions []model.Interruption
	for at := 30.0; at < clipDuration; at += 30 {
		interruptions = append(interruptions, model.Interruption{At: at, Duration: bannerDuration})
	}
	return interruptions, nil
}
