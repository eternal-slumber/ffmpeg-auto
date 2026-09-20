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

func (f FFmpeg) Render(ctx context.Context, sourcePath, bannerPath, outputPath string, clip model.ClipPlan) error {
	if sourcePath == "" {
		return fmt.Errorf("source path is required")
	}
	if bannerPath == "" {
		return fmt.Errorf("banner path is required")
	}
	if outputPath == "" {
		return fmt.Errorf("output path is required")
	}
	if err := validateClip(clip); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	command := f.Command
	if command == "" {
		command = "ffmpeg"
	}
	out, err := exec.CommandContext(ctx, command, renderArgs(sourcePath, bannerPath, outputPath, clip)...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg clip %d: %w: %s", clip.Number, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func validateClip(clip model.ClipPlan) error {
	if clip.Start < 0 || math.IsNaN(clip.Start) || math.IsInf(clip.Start, 0) {
		return fmt.Errorf("clip start must be a finite non-negative number, got %v", clip.Start)
	}
	if clip.Duration <= 0 || math.IsNaN(clip.Duration) || math.IsInf(clip.Duration, 0) {
		return fmt.Errorf("clip duration must be a finite positive number, got %v", clip.Duration)
	}
	if len(clip.Interruptions) == 0 {
		return fmt.Errorf("clip must contain at least one interruption")
	}
	previous := 0.0
	for i, interruption := range clip.Interruptions {
		if interruption.At <= 0 || interruption.At >= clip.Duration || math.IsNaN(interruption.At) || math.IsInf(interruption.At, 0) {
			return fmt.Errorf("interruption %d position must be inside the clip, got %v", i+1, interruption.At)
		}
		if i > 0 && interruption.At <= previous {
			return fmt.Errorf("interruption %d must be after interruption %d", i+1, i)
		}
		if interruption.Duration <= 0 || math.IsNaN(interruption.Duration) || math.IsInf(interruption.Duration, 0) {
			return fmt.Errorf("interruption %d duration must be a finite positive number, got %v", i+1, interruption.Duration)
		}
		previous = interruption.At
	}
	return nil
}

func renderArgs(sourcePath, bannerPath, outputPath string, clip model.ClipPlan) []string {
	return []string{
		"-y",
		"-v", "error",
		"-ss", seconds(clip.Start),
		"-t", seconds(clip.Duration),
		"-i", sourcePath,
		"-i", bannerPath,
		"-filter_complex", renderFilter(clip),
		"-map", "[video]",
		"-map", "[audio]",
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "18",
		"-pix_fmt", "yuv420p",
		"-c:a", "aac",
		"-movflags", "+faststart",
		outputPath,
	}
}

func renderFilter(clip model.ClipPlan) string {
	interruptions := clip.Interruptions
	filters := []string{
		"[0:v:0]setpts=PTS-STARTPTS,split=2[bgsrc][fgsrc]",
		"[bgsrc]crop='min(iw,ih*9/16)':'min(ih,iw*16/9)',scale=1080:1920,setsar=1,boxblur=20:1[bg]",
		"[fgsrc]scale=1080:1920:force_original_aspect_ratio=decrease:force_divisible_by=2,setsar=1[fg]",
		"[bg][fg]overlay=(W-w)/2:(H-h)/2,format=yuv420p[base]",
	}

	videoInputs := make([]string, 0, len(interruptions)*2+1)
	for i := 0; i <= len(interruptions); i++ {
		videoInputs = append(videoInputs, fmt.Sprintf("[sv%d]", i))
	}
	for i := range interruptions {
		videoInputs = append(videoInputs, fmt.Sprintf("[sf%d]", i))
	}
	filters = append(filters, fmt.Sprintf("[base]split=%d%s", len(videoInputs), strings.Join(videoInputs, "")))

	audioInputs := make([]string, len(interruptions)+1)
	for i := range audioInputs {
		audioInputs[i] = fmt.Sprintf("[sa%d]", i)
	}
	filters = append(filters, fmt.Sprintf("[0:a:0]asetpts=PTS-STARTPTS,asplit=%d%s", len(audioInputs), strings.Join(audioInputs, "")))

	bannerVideoInputs := make([]string, len(interruptions))
	bannerAudioInputs := make([]string, len(interruptions))
	if len(interruptions) == 1 {
		bannerVideoInputs[0] = "[1:v:0]"
		bannerAudioInputs[0] = "[1:a:0]"
	} else {
		for i := range interruptions {
			bannerVideoInputs[i] = fmt.Sprintf("[bv%d]", i)
			bannerAudioInputs[i] = fmt.Sprintf("[ba%d]", i)
		}
		filters = append(filters,
			fmt.Sprintf("[1:v:0]split=%d%s", len(interruptions), strings.Join(bannerVideoInputs, "")),
			fmt.Sprintf("[1:a:0]asplit=%d%s", len(interruptions), strings.Join(bannerAudioInputs, "")),
		)
	}

	videoParts := make([]string, 0, len(interruptions)*2+1)
	audioParts := make([]string, 0, len(interruptions)*2+1)
	start := 0.0
	for i := 0; i <= len(interruptions); i++ {
		end := clip.Duration
		if i < len(interruptions) {
			end = interruptions[i].At
		}
		filters = append(filters,
			fmt.Sprintf("[sv%d]trim=start=%s:end=%s,setpts=PTS-STARTPTS[vseg%d]", i, seconds(start), seconds(end), i),
			fmt.Sprintf("[sa%d]atrim=start=%s:end=%s,asetpts=PTS-STARTPTS,aformat=sample_rates=48000:channel_layouts=stereo[aseg%d]", i, seconds(start), seconds(end), i),
		)
		videoParts = append(videoParts, fmt.Sprintf("[vseg%d]", i))
		audioParts = append(audioParts, fmt.Sprintf("[aseg%d]", i))

		if i == len(interruptions) {
			continue
		}
		interruption := interruptions[i]
		// ponytail: wide banners are contained at full frame width; add crop-aware artwork layouts if 45% frame area becomes mandatory.
		filters = append(filters,
			fmt.Sprintf("[sf%d]trim=start=%s,trim=end_frame=1,setpts=PTS-STARTPTS,tpad=stop_mode=clone:stop_duration=%s,trim=duration=%s[freeze%d]", i, seconds(interruption.At), seconds(interruption.Duration), seconds(interruption.Duration), i),
			fmt.Sprintf("%strim=duration=%s,setpts=PTS-STARTPTS,chromakey=0x00FF00:0.30:0.10,scale=1080:1920:force_original_aspect_ratio=decrease:force_divisible_by=2[ban%d]", bannerVideoInputs[i], seconds(interruption.Duration), i),
			fmt.Sprintf("[freeze%d][ban%d]overlay=(W-w)/2:(H-h)/2:shortest=1,format=yuv420p[vint%d]", i, i, i),
			fmt.Sprintf("%satrim=duration=%s,asetpts=PTS-STARTPTS,aformat=sample_rates=48000:channel_layouts=stereo[aint%d]", bannerAudioInputs[i], seconds(interruption.Duration), i),
		)
		videoParts = append(videoParts, fmt.Sprintf("[vint%d]", i))
		audioParts = append(audioParts, fmt.Sprintf("[aint%d]", i))
		start = interruption.At
	}

	filters = append(filters,
		fmt.Sprintf("%sconcat=n=%d:v=1:a=0[video]", strings.Join(videoParts, ""), len(videoParts)),
		fmt.Sprintf("%sconcat=n=%d:v=0:a=1[audio]", strings.Join(audioParts, ""), len(audioParts)),
	)
	return strings.Join(filters, ";")
}

func seconds(value float64) string {
	return strconv.FormatFloat(value, 'f', 6, 64)
}
