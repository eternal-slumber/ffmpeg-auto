package model

type MediaMetadata struct {
	Duration   float64 `json:"duration_seconds"`
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	FPS        float64 `json:"fps"`
	VideoCodec string  `json:"video_codec,omitempty"`
	AudioCodec string  `json:"audio_codec,omitempty"`
	HasVideo   bool    `json:"has_video"`
	HasAudio   bool    `json:"has_audio"`
}
