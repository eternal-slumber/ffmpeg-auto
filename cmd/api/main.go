package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"content-factory/internal/media"
	"content-factory/internal/model"
	"content-factory/internal/planner"
	"content-factory/internal/renderer"
)

func main() {
	sourcePath := "storage/incoming/source.mp4"
	if len(os.Args) > 1 {
		sourcePath = os.Args[1]
	}
	parts := 10
	if len(os.Args) > 2 {
		var err error
		parts, err = strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid parts %q: %v\n", os.Args[2], err)
			os.Exit(1)
		}
	}
	bannerPath := "storage/incoming/banner.mp4"
	if len(os.Args) > 3 {
		bannerPath = os.Args[3]
	}
	outputPath := "storage/output/clip-001.mp4"
	if len(os.Args) > 4 {
		outputPath = os.Args[4]
	}

	ctx := context.Background()
	prober := media.FFProbe{}
	source, err := prober.Probe(ctx, sourcePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	banner, err := prober.Probe(ctx, bannerPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	clips, err := planner.EqualSplit(source.Duration, parts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for i := range clips {
		clips[i].Interruptions, err = planner.PlanInterruptions(clips[i].Duration, banner.Duration)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if err := (renderer.FFmpeg{}).Render(ctx, sourcePath, outputPath, clips[0]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	rendered, err := prober.Probe(ctx, outputPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	result := struct {
		Source       model.MediaMetadata `json:"source"`
		Banner       model.MediaMetadata `json:"banner"`
		Clips        []model.ClipPlan    `json:"clips"`
		RenderedPath string              `json:"rendered_path"`
		Rendered     model.MediaMetadata `json:"rendered"`
	}{source, banner, clips, outputPath, rendered}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
