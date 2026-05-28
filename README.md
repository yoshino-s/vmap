# vmap

A simple vsock scanner built with Cobra.

## Features

- Cobra-based CLI
- `--mode [host|guest|auto]`, default `auto`
- in `auto` mode, runtime detection picks `host` or `guest`; detected mode is applied and logged
- `--cid` supports list/range/all (`1,2,3`, `1-10`, `all`)
- `--port` supports list/range/all (`1,2,3`, `1-10`, `all`), default `all`
- default behavior checks connectivity only
- `--detect` first completes connectivity scan, then probes only open targets
- `--detect` sends payloads and prints non-empty responses
- terminal progress bar for scan progress
- connectivity results are replayed in a full summary after connectivity phase
- `--timeout` and `--interval` are optional, default disabled
- `--log-level` controls verbosity: `debug`, `info`, `warn`, `error`
- `CGO_ENABLED=0` compatible builds

## Installation

```bash
go build -o vmap .
```

## Usage

```bash
./vmap [flags]
```

### Flags

- `--mode string` scan mode: `host`, `guest`, `auto` (default `auto`)
- `--cid string` CID list/range/all
- `--port string` port list/range/all (default `all`)
- `--detect` send common payloads and print non-empty responses
- `--timeout duration` per-connection timeout, `0` means no timeout
- `--interval duration` sleep interval between probes, `0` means no interval
- `--log-level string` log level: `debug`, `info`, `warn`, `error` (default `info`)

### CID behavior when `--cid` is empty

- host mode: scan `all`
- guest mode: scan CID `2` and local CID
- auto mode: detect runtime role first; if detection fails, fallback to CID `all`
- if local CID cannot be retrieved in guest mode: scan `all`

### Detect payloads

When `--detect` is enabled, scanner first finishes connectivity checks for all targets, then tests only open ports with:

1. empty payload
2. empty JSON (`{}`)
3. single newline (`\n`)
4. single digit (`1`)
5. single letter (`a`)
6. random UUID

Any non-empty response is printed.

### Logging

- `info` (default): stage/status, open targets, detect hits, summary
- `warn`: fallback warnings (for example, local CID unavailable)
- `error`: reserved for fatal-level messages
- `debug`: includes detailed probe traces, including detect request payload and response bytes

## Notes

- this tool supports Linux runtime for actual vsock probing
- non-Linux builds are provided for packaging compatibility but return an unsupported error at runtime
- `all` CID range in this tool is `0-65535`
- `all` port range is `1-65535`

## Release

Tag pushes like `v1.0.0` trigger GitHub Actions to build and publish release artifacts for:

- linux/amd64
- linux/arm64
- darwin/amd64
- darwin/arm64

For each platform, two artifact variants are published:

- Standard: `vmap_<version>_<os>_<arch>.tar.gz`
- Thin: `vmap_<version>_<os>_<arch>_thin.tar.gz`

Thin artifacts are optimized for size using:

- `CGO_ENABLED=0`
- stripped symbols (`-s -w`)
- empty build id (`-buildid=`)
- gzip archive format (`tar.gz`)
