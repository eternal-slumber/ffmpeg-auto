package renderer

import (
	"context"
	"math"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"content-factory/internal/model"
)

func TestRenderArgs(t *testing.T) {
	clip := model.ClipPlan{
		Number:   2,
		Start:    3.417,
		Duration: 3.417,
		Interruptions: []model.Interruption{
			{At: 1.7085, Duration: 3.833333},
		},
	}
	args := renderArgs("source video.mp4", "banner video.mp4", "output clip.mp4", clip)
	for _, want := range []string{"source video.mp4", "banner video.mp4", "output clip.mp4", "3.417000"} {
		if !slices.Contains(args, want) {
			t.Fatalf("arguments do not contain %q: %#v", want, args)
		}
	}
	filter := renderFilter(clip)
	for _, want := range []string{
		"chromakey=0x00FF00:0.30:0.10",
		"tpad=stop_mode=clone:stop_duration=3.833333",
		"[vseg0][vint0][vseg1]concat=n=3:v=1:a=0[video]",
		"[aseg0][aint0][aseg1]concat=n=3:v=0:a=1[audio]",
	} {
		if !strings.Contains(filter, want) {
			t.Fatalf("filter does not contain %q:\n%s", want, filter)
		}
	}
}

func TestRenderFilterSupportsMultipleInterruptions(t *testing.T) {
	filter := renderFilter(model.ClipPlan{
		Duration: 130,
		Interruptions: []model.Interruption{
			{At: 30, Duration: 5},
			{At: 60, Duration: 5},
			{At: 90, Duration: 5},
			{At: 120, Duration: 5},
		},
	})
	for _, want := range []string{
		"[base]split=9",
		"[1:v:0]split=4",
		"[1:a:0]asplit=4",
		"concat=n=9:v=1:a=0[video]",
		"concat=n=9:v=0:a=1[audio]",
	} {
		if !strings.Contains(filter, want) {
			t.Fatalf("filter does not contain %q:\n%s", want, filter)
		}
	}
}

func TestValidateClipRejectsInvalidPlan(t *testing.T) {
	tests := []model.ClipPlan{
		{Start: -1, Duration: 10, Interruptions: []model.Interruption{{At: 5, Duration: 1}}},
		{Start: math.NaN(), Duration: 10, Interruptions: []model.Interruption{{At: 5, Duration: 1}}},
		{Duration: 0, Interruptions: []model.Interruption{{At: 5, Duration: 1}}},
		{Duration: 10},
		{Duration: 10, Interruptions: []model.Interruption{{At: 0, Duration: 1}}},
		{Duration: 10, Interruptions: []model.Interruption{{At: 10, Duration: 1}}},
		{Duration: 10, Interruptions: []model.Interruption{{At: 5, Duration: 0}}},
		{Duration: 10, Interruptions: []model.Interruption{{At: 6, Duration: 1}, {At: 5, Duration: 1}}},
	}
	for _, clip := range tests {
		if err := validateClip(clip); err == nil {
			t.Fatalf("validateClip(%+v) returned no error", clip)
		}
	}
}

func TestFFmpegRenderMultipleInterruptions(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	for _, command := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(command); err != nil {
			t.Skipf("%s is not installed", command)
		}
	}

	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "source.mp4")
	bannerPath := filepath.Join(dir, "banner.mp4")
	outputPath := filepath.Join(dir, "output.mp4")
	run := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
			t.Fatalf("ffmpeg fixture: %v: %s", err, out)
		}
	}
	run("-y", "-v", "error",
		"-f", "lavfi", "-i", "testsrc2=size=160x90:rate=10:duration=2",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=48000:duration=2",
		"-shortest", "-c:v", "libx264", "-pix_fmt", "yuv420p", "-c:a", "aac", sourcePath)
	run("-y", "-v", "error",
		"-f", "lavfi", "-i", "color=c=0x00FF00:size=160x60:rate=10:duration=0.3,drawbox=x=40:y=15:w=80:h=30:color=red:t=fill",
		"-f", "lavfi", "-i", "sine=frequency=880:sample_rate=48000:duration=0.3",
		"-shortest", "-c:v", "libx264", "-pix_fmt", "yuv420p", "-c:a", "aac", bannerPath)

	clip := model.ClipPlan{
		Number:   1,
		Duration: 2,
		Interruptions: []model.Interruption{
			{At: 0.5, Duration: 0.3},
			{At: 1.5, Duration: 0.3},
		},
	}
	if err := (FFmpeg{}).Render(context.Background(), sourcePath, bannerPath, outputPath, clip); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=nw=1:nk=1", outputPath).Output()
	if err != nil {
		t.Fatal(err)
	}
	duration, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(duration-2.6) > 0.15 {
		t.Fatalf("output duration = %v, want about 2.6", duration)
	}
}
