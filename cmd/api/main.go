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

	prober := media.FFProbe{}
	source, err := prober.Probe(context.Background(), sourcePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	banner, err := prober.Probe(context.Background(), bannerPath)
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

	result := struct {
		Source model.MediaMetadata `json:"source"`
		Banner model.MediaMetadata `json:"banner"`
		Clips  []model.ClipPlan    `json:"clips"`
	}{source, banner, clips}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
