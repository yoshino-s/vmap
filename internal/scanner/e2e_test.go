//go:build linux

package scanner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
	"vmap/internal/vsock"
)

func TestE2E_DetectLocalVsockListener(t *testing.T) {
	localCID, err := vsock.LocalCID()
	if err != nil {
		t.Skipf("skip e2e: local vsock CID unavailable: %v", err)
	}

	port, cleanup, err := startLocalVSockListener()
	if err != nil {
		t.Skipf("skip e2e: cannot start local vsock listener: %v", err)
	}
	defer cleanup()

	output := captureStdout(t, func() {
		s, err := New(Options{
			Mode:      modeAuto,
			CIDInput:  fmt.Sprintf("%d", localCID),
			PortInput: fmt.Sprintf("%d", port),
			Timeout:   1 * time.Second,
		})
		if err != nil {
			t.Fatalf("create scanner failed: %v", err)
		}

		if err := s.Run(context.Background()); err != nil {
			t.Fatalf("scanner run failed: %v", err)
		}
	})

	openLog := fmt.Sprintf("[open] %d:%d", localCID, port)
	if !strings.Contains(output, openLog) {
		t.Fatalf("expected output to contain %q, got:\n%s", openLog, output)
	}

	if !strings.Contains(output, "overriding mode") {
		t.Fatalf("expected auto mode override log, got:\n%s", output)
	}
}

func startLocalVSockListener() (uint32, func(), error) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < 64; i++ {
		port := uint32(20000 + rng.Intn(30000))

		fd, err := unix.Socket(unix.AF_VSOCK, unix.SOCK_STREAM, 0)
		if err != nil {
			return 0, nil, err
		}

		addr := &unix.SockaddrVM{CID: unix.VMADDR_CID_ANY, Port: port}
		if err := unix.Bind(fd, addr); err != nil {
			_ = unix.Close(fd)
			if errors.Is(err, unix.EADDRINUSE) {
				continue
			}
			return 0, nil, err
		}

		if err := unix.Listen(fd, 8); err != nil {
			_ = unix.Close(fd)
			return 0, nil, err
		}

		done := make(chan struct{})
		go func() {
			defer close(done)
			for {
				connFD, _, err := unix.Accept(fd)
				if err != nil {
					if errors.Is(err, unix.EINTR) {
						continue
					}
					return
				}
				_ = unix.Close(connFD)
			}
		}()

		cleanup := func() {
			_ = unix.Close(fd)
			<-done
		}

		return port, cleanup, nil
	}

	return 0, nil, errors.New("no available vsock port")
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe failed: %v", err)
	}

	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
		_ = r.Close()
		_ = w.Close()
	}()

	fn()
	_ = w.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("read captured stdout failed: %v", err)
	}

	return buf.String()
}
