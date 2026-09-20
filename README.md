# Content Factory

Stage 1 reads media metadata through the system `ffprobe` executable. Stage 2
splits the detected duration into an equal clip plan. Stage 3 probes the banner
and adds its insertion points to every clip. No video is rendered yet.

Requirements: Go 1.22+ and FFmpeg/ffprobe available in `PATH` on macOS or Linux.

```sh
go test ./...
go run ./cmd/api
# or: go run ./cmd/api /path/to/video.mp4 10 /path/to/banner.mp4
```

Defaults: `storage/incoming/source.mp4`, 10 clips, and
`storage/incoming/banner.mp4`.

See `PROJECT.md` for the end-to-end description and development log.
