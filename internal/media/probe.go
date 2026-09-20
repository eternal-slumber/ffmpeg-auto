package media

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"content-factory/internal/model"
)

type Prober interface {
	Probe(context.Context, string) (model.MediaMetadata, error)
}

type FFProbe struct {
	Command string
}

func (p FFProbe) Probe(ctx context.Context, path string) (model.MediaMetadata, error) {
	command := p.Command
	if command == "" {
		command = "ffprobe"
	}

	out, err := exec.CommandContext(ctx, command,
		"-v", "error",
		"-show_entries", "format=duration:stream=codec_type,codec_name,width,height,avg_frame_rate,r_frame_rate",
		"-of", "json",
		path,
	).CombinedOutput()
	if err != nil {
		return model.MediaMetadata{}, fmt.Errorf("ffprobe %q: %w: %s", path, err, strings.TrimSpace(string(out)))
	}

	metadata, err := parseProbeOutput(out)
	if err != nil {
		return model.MediaMetadata{}, fmt.Errorf("parse ffprobe output for %q: %w", path, err)
	}
	return metadata, nil
}

type probeOutput struct {
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
	Streams []struct {
		CodecType    string `json:"codec_type"`
		CodecName    string `json:"codec_name"`
		Width        int    `json:"width"`
		Height       int    `json:"height"`
		AvgFrameRate string `json:"avg_frame_rate"`
		RFrameRate   string `json:"r_frame_rate"`
	} `json:"streams"`
}

func parseProbeOutput(data []byte) (model.MediaMetadata, error) {
	var raw probeOutput
	if err := json.Unmarshal(data, &raw); err != nil {
		return model.MediaMetadata{}, err
	}

	var metadata model.MediaMetadata
	if raw.Format.Duration != "" {
		duration, err := strconv.ParseFloat(raw.Format.Duration, 64)
		if err != nil {
			return model.MediaMetadata{}, fmt.Errorf("invalid duration %q: %w", raw.Format.Duration, err)
		}
		metadata.Duration = duration
	}

	for _, stream := range raw.Streams {
		switch stream.CodecType {
		case "video":
			if metadata.HasVideo {
				continue
			}
			metadata.HasVideo = true
			metadata.VideoCodec = stream.CodecName
			metadata.Width = stream.Width
			metadata.Height = stream.Height
			fps := stream.AvgFrameRate
			if fps == "" || fps == "0/0" {
				fps = stream.RFrameRate
			}
			if fps != "" && fps != "0/0" {
				var err error
				metadata.FPS, err = parseRate(fps)
				if err != nil {
					return model.MediaMetadata{}, err
				}
			}
		case "audio":
			if !metadata.HasAudio {
				metadata.HasAudio = true
				metadata.AudioCodec = stream.CodecName
			}
		}
	}

	return metadata, nil
}

func parseRate(rate string) (float64, error) {
	parts := strings.Split(rate, "/")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid frame rate %q", rate)
	}
	numerator, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid frame rate %q: %w", rate, err)
	}
	denominator, err := strconv.ParseFloat(parts[1], 64)
	if err != nil || denominator == 0 {
		return 0, fmt.Errorf("invalid frame rate %q", rate)
	}
	return numerator / denominator, nil
}
