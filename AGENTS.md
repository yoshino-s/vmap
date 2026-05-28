## Cursor Cloud specific instructions

This is a Go CLI tool (`vmap`) — a vsock port scanner. No external services, databases, or containers are required.

### Quick reference

| Task | Command |
|------|---------|
| Install deps | `go mod download` |
| Run tests | `go test ./...` |
| Lint | `go vet ./...` |
| Build | `CGO_ENABLED=0 go build -o vmap .` |
| Run | `./vmap --help` |

### Notes

- Go 1.25.0 is required (see `go.mod`).
- The binary uses Linux `AF_VSOCK` sockets. Actual vsock scanning only works on Linux hosts with the `vhost_vsock` kernel module. In Cloud Agent VMs, vsock peers are unavailable, so scans complete instantly with no open ports found — this is expected.
- Unit tests are pure logic tests (range parsing, CID resolution) and do not require vsock hardware.
- No linter beyond `go vet` is configured in the repo.
