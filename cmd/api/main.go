package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"content-factory/internal/factory"
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

	result, err := factory.Run(context.Background(), sourcePath, bannerPath, outputPath, parts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
