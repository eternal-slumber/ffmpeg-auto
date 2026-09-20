# Content Factory

Stage 1 reads media metadata through the system `ffprobe` executable. Stage 2
splits the detected duration into an equal clip plan. Stage 3 probes the banner
and adds its insertion points to every clip. Stage 4 renders the first planned
clip to MP4 and verifies it with `ffprobe`. Stage 5 normalizes that clip to
TikTok-ready `1080x1920` with a blurred background. Stage 6 inserts the banner
with chroma key, freeze-frame, banner audio, and source resume.

Requirements: Go 1.22+ and FFmpeg/ffprobe available in `PATH` on macOS or Linux.

```sh
go test ./...
go run ./cmd/api
# or: go run ./cmd/api /path/to/video.mp4 10 /path/to/banner.mp4 /path/to/output.mp4
```

Defaults: `storage/incoming/source.mp4`, 10 clips, and
`storage/incoming/banner.mp4`. The first clip is written to
`storage/output/clip-001.mp4`.

See `PROJECT.md` for the end-to-end description and development log.
