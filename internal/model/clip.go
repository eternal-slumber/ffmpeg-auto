package model

type ClipPlan struct {
	Number        int            `json:"number"`
	Start         float64        `json:"start_seconds"`
	Duration      float64        `json:"duration_seconds"`
	Interruptions []Interruption `json:"interruptions,omitempty"`
}
