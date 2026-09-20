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
		"-filter_complex", "[0:v:0]split=2[bgsrc][fgsrc];[bgsrc]crop='min(iw,ih*9/16)':'min(ih,iw*16/9)',scale=1080:1920,setsar=1,boxblur=20:1[bg];[fgsrc]scale=1080:1920:force_original_aspect_ratio=decrease:force_divisible_by=2,setsar=1[fg];[bg][fg]overlay=(W-w)/2:(H-h)/2,format=yuv420p[video]",
		"-map", "[video]",
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
