package factory

import (
	"context"
	"fmt"

	"content-factory/internal/media"
	"content-factory/internal/model"
	"content-factory/internal/planner"
	"content-factory/internal/renderer"
)

type Result struct {
	Source       model.MediaMetadata `json:"source"`
	Banner       model.MediaMetadata `json:"banner"`
	Clips        []model.ClipPlan    `json:"clips"`
	RenderedPath string              `json:"rendered_path"`
	Rendered     model.MediaMetadata `json:"rendered"`
}

func Run(ctx context.Context, sourcePath, bannerPath, outputPath string, parts int) (Result, error) {
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
	if err := (renderer.FFmpeg{}).Render(ctx, sourcePath, bannerPath, outputPath, clips[0]); err != nil {
		return Result{}, err
	}
	rendered, err := prober.Probe(ctx, outputPath)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Source:       source,
		Banner:       banner,
		Clips:        clips,
		RenderedPath: outputPath,
		Rendered:     rendered,
	}, nil
}
