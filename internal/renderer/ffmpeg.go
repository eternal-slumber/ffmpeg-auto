package renderer

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"content-factory/internal/model"
)

type FFmpeg struct {
	Command string
}

func (f FFmpeg) Render(ctx context.Context, sourcePath, outputPath string, clip model.ClipPlan) error {
	if sourcePath == "" {
		return fmt.Errorf("source path is required")
	}
	if outputPath == "" {
		return fmt.Errorf("output path is required")
	}
	if clip.Start < 0 || math.IsNaN(clip.Start) || math.IsInf(clip.Start, 0) {
		return fmt.Errorf("clip start must be a finite non-negative number, got %v", clip.Start)
	}
	if clip.Duration <= 0 || math.IsNaN(clip.Duration) || math.IsInf(clip.Duration, 0) {
		return fmt.Errorf("clip duration must be a finite positive number, got %v", clip.Duration)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	command := f.Command
	if command == "" {
		command = "ffmpeg"
	}
	out, err := exec.CommandContext(ctx, command, renderArgs(sourcePath, outputPath, clip)...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg clip %d: %w: %s", clip.Number, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func renderArgs(sourcePath, outputPath string, clip model.ClipPlan) []string {
	seconds := func(value float64) string {
		return strconv.FormatFloat(value, 'f', 6, 64)
	}
	return []string{
		"-y",
		"-v", "error",
		"-i", sourcePath,
		"-ss", seconds(clip.Start),
		"-t", seconds(clip.Duration),
		"-filter_complex", "[0:v:0]split=2[bgsrc][fgsrc];[bgsrc]crop='min(iw,ih*9/16)':'min(ih,iw*16/9)',scale=1080:1920,setsar=1,boxblur=20:1[bg];[fgsrc]scale=1080:1920:force_original_aspect_ratio=decrease:force_divisible_by=2,setsar=1[fg];[bg][fg]overlay=(W-w)/2:(H-h)/2,format=yuv420p[video]",
		"-map", "[video]",
		"-map", "0:a:0?",
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "18",
		"-pix_fmt", "yuv420p",
		"-c:a", "aac",
		"-movflags", "+faststart",
		outputPath,
	}
}
