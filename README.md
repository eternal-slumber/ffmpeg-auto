# Content Factory

The application probes source and banner media, splits the source into equal
parts, inserts the banner with chroma key and a source freeze, and renders every
planned clip as a TikTok-ready `1080x1920` MP4.

Requirements: Go 1.22+ and FFmpeg/ffprobe available in `PATH` on macOS or Linux.

```sh
go test ./...
go run ./cmd/api
# or: go run ./cmd/api /path/to/video.mp4 10 /path/to/banner.mp4 /path/to/output.mp4
```

Debug UI:

```sh
go run ./cmd/web
# open http://127.0.0.1:8080
```

Defaults: `storage/incoming/source.mp4`, 10 clips, and
`storage/incoming/banner.mp4`. Results are written to
`storage/output/clip-001.mp4` through `clip-010.mp4`. The debug UI previews all
rendered clips and shows live batch progress.

See `PROJECT.md` for the end-to-end description and development log.
