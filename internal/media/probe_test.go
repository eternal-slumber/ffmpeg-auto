package media

import (
	"math"
	"testing"
)

func TestParseProbeOutput(t *testing.T) {
	data := []byte(`{
  "streams": [
    {"codec_type":"video","codec_name":"h264","width":1920,"height":1080,"avg_frame_rate":"30000/1001"},
    {"codec_type":"audio","codec_name":"aac"}
  ],
  "format": {"duration":"23.823000"}
}`)

	got, err := parseProbeOutput(data)
	if err != nil {
		t.Fatal(err)
	}
	if !got.HasVideo || !got.HasAudio || got.VideoCodec != "h264" || got.AudioCodec != "aac" {
		t.Fatalf("unexpected streams: %+v", got)
	}
	if got.Width != 1920 || got.Height != 1080 || math.Abs(got.FPS-29.97003) > 0.00001 || got.Duration != 23.823 {
		t.Fatalf("unexpected metadata: %+v", got)
	}
}

func TestParseProbeOutputFallsBackToRealFrameRate(t *testing.T) {
	data := []byte(`{"streams":[{"codec_type":"video","r_frame_rate":"25/1"}],"format":{}}`)

	got, err := parseProbeOutput(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.FPS != 25 || got.HasAudio {
		t.Fatalf("unexpected metadata: %+v", got)
	}
}

func TestParseProbeOutputRejectsInvalidDuration(t *testing.T) {
	_, err := parseProbeOutput([]byte(`{"format":{"duration":"nope"}}`))
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestParseProbeOutputRejectsInvalidFrameRate(t *testing.T) {
	_, err := parseProbeOutput([]byte(`{"streams":[{"codec_type":"video","avg_frame_rate":"nope"}]}`))
	if err == nil {
		t.Fatal("expected an error")
	}
}
