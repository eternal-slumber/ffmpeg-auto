package factory

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"content-factory/internal/media"
	"content-factory/internal/model"
	"content-factory/internal/planner"
	"content-factory/internal/renderer"
)

type Result struct {
	Source  model.MediaMetadata `json:"source"`
	Banner  model.MediaMetadata `json:"banner"`
	Clips   []model.ClipPlan    `json:"clips"`
	Outputs []Output            `json:"outputs"`
}

type Output struct {
	Number   int                 `json:"number"`
	Path     string              `json:"path"`
	Metadata model.MediaMetadata `json:"metadata"`
}

func Run(ctx context.Context, sourcePath, bannerPath, outputPath string, parts int) (Result, error) {
	return RunWithProgress(ctx, sourcePath, bannerPath, outputPath, parts, nil)
}

func RunWithProgress(ctx context.Context, sourcePath, bannerPath, outputPath string, parts int, progress func(done, total int)) (Result, error) {
	prober := media.FFProbe{}
	source, err := prober.Probe(ctx, sourcePath)
	if err != nil {
		return Result{}, err
	}
	banner, err := prober.Probe(ctx, bannerPath)
	if err != nil {
		return Result{}, err
	}
	if !source.HasAudio || !banner.HasAudio {
		return Result{}, fmt.Errorf("source and banner must contain audio")
	}
	clips, err := planner.EqualSplit(source.Duration, parts)
	if err != nil {
		return Result{}, err
	}
	for i := range clips {
		clips[i].Interruptions, err = planner.PlanInterruptions(clips[i].Duration, banner.Duration)
		if err != nil {
			return Result{}, err
		}
	}
	outputs := make([]Output, len(clips))
	ffmpeg := renderer.FFmpeg{}
	for i, clip := range clips {
		path := numberedOutputPath(outputPath, clip.Number)
		if err := ffmpeg.Render(ctx, sourcePath, bannerPath, path, clip); err != nil {
			return Result{}, fmt.Errorf("render clip %d: %w", clip.Number, err)
		}
		metadata, err := prober.Probe(ctx, path)
		if err != nil {
			return Result{}, fmt.Errorf("probe clip %d: %w", clip.Number, err)
		}
		outputs[i] = Output{Number: clip.Number, Path: path, Metadata: metadata}
		if progress != nil {
			progress(i+1, len(clips))
		}
	}
	return Result{
		Source:  source,
		Banner:  banner,
		Clips:   clips,
		Outputs: outputs,
	}, nil
}

func numberedOutputPath(firstPath string, number int) string {
	if number == 1 {
		return firstPath
	}
	ext := filepath.Ext(firstPath)
	stem := strings.TrimSuffix(filepath.Base(firstPath), ext)
	stem = strings.TrimSuffix(stem, "-001")
	return filepath.Join(filepath.Dir(firstPath), fmt.Sprintf("%s-%03d%s", stem, number, ext))
}
