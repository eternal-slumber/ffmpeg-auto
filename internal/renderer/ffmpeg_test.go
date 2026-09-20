package renderer

import (
	"reflect"
	"testing"

	"content-factory/internal/model"
)

func TestRenderArgs(t *testing.T) {
	got := renderArgs("source video.mp4", "output clip.mp4", model.ClipPlan{
		Number:   2,
		Start:    3.417,
		Duration: 3.417,
	})
	want := []string{
		"-y",
		"-v", "error",
		"-i", "source video.mp4",
		"-ss", "3.417000",
		"-t", "3.417000",
		"-map", "0:v:0",
		"-map", "0:a:0?",
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "18",
		"-pix_fmt", "yuv420p",
		"-c:a", "aac",
		"-movflags", "+faststart",
		"output clip.mp4",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("renderArgs() = %#v, want %#v", got, want)
	}
}
