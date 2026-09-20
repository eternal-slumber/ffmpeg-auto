package model

type Interruption struct {
	At       float64 `json:"at_seconds"`
	Duration float64 `json:"duration_seconds"`
}
