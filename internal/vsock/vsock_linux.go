//go:build linux

package vsock

import (
	"errors"
	"fmt"
	"io"
	"net"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

const HostCID uint32 = unix.VMADDR_CID_HOST

type Addr struct {
	CID  uint32
	Port uint32
}

func (a Addr) Network() string { return "vsock" }

func (a Addr) String() string {
	return fmt.Sprintf("%d:%d", a.CID, a.Port)
}

type Conn struct {
	fd         int
	localCID   uint32
	remoteCID  uint32
	remotePort uint32
	closed     bool
}

func Dial(cid uint32, port uint32, timeout time.Duration) (*Conn, error) {
	fd, err := unix.Socket(unix.AF_VSOCK, unix.SOCK_STREAM, 0)
	if err != nil {
		return nil, fmt.Errorf("create vsock socket: %w", err)
	}

	if timeout > 0 {
		if err := connectWithTimeout(fd, cid, port, timeout); err != nil {
			_ = unix.Close(fd)
			return nil, err
		}
		_ = applyReadWriteTimeout(fd, timeout)
	} else {
		if err := unix.Connect(fd, &unix.SockaddrVM{CID: cid, Port: port}); err != nil {
			_ = unix.Close(fd)
			return nil, fmt.Errorf("connect to %d:%d failed: %w", cid, port, err)
		}
	}

	localCID, err := LocalCID()
	if err != nil {
		localCID = 0
	}

	return &Conn{fd: fd, localCID: localCID, remoteCID: cid, remotePort: port}, nil
}

func LocalCID() (uint32, error) {
	fd, err := unix.Socket(unix.AF_VSOCK, unix.SOCK_DGRAM, 0)
	if err != nil {
		return 0, fmt.Errorf("create vsock socket: %w", err)
	}
	defer unix.Close(fd)

	cid, err := unix.IoctlGetUint32(fd, unix.IOCTL_VM_SOCKETS_GET_LOCAL_CID)
	if err != nil {
		return 0, fmt.Errorf("get local CID: %w", err)
	}
	return cid, nil
}

func (c *Conn) Read(p []byte) (int, error) {
	if c.closed {
		return 0, io.ErrClosedPipe
	}
	n, err := unix.Read(c.fd, p)
	if err != nil {
		return 0, osOrTimeoutError(err)
	}
	if n == 0 {
		return 0, io.EOF
	}
	return n, nil
}

func (c *Conn) Write(p []byte) (int, error) {
	if c.closed {
		return 0, io.ErrClosedPipe
	}
	n, err := unix.Write(c.fd, p)
	if err != nil {
		return n, osOrTimeoutError(err)
	}
	return n, nil
}

func (c *Conn) Close() error {
	if c.closed {
		return nil
	}
	c.closed = true
	if err := unix.Close(c.fd); err != nil {
		return fmt.Errorf("close vsock connection: %w", err)
	}
	return nil
}

func (c *Conn) LocalAddr() net.Addr {
	return Addr{CID: c.localCID, Port: 0}
}

func (c *Conn) RemoteAddr() net.Addr {
	return Addr{CID: c.remoteCID, Port: c.remotePort}
}

func (c *Conn) SetDeadline(t time.Time) error {
	if err := c.SetReadDeadline(t); err != nil {
		return err
	}
	return c.SetWriteDeadline(t)
}

func (c *Conn) SetReadDeadline(t time.Time) error {
	if c.closed {
		return io.ErrClosedPipe
	}
	timeout := timeUntil(t)
	return applySockoptTimeout(c.fd, unix.SO_RCVTIMEO, timeout)
}

func (c *Conn) SetWriteDeadline(t time.Time) error {
	if c.closed {
		return io.ErrClosedPipe
	}
	timeout := timeUntil(t)
	return applySockoptTimeout(c.fd, unix.SO_SNDTIMEO, timeout)
}

func connectWithTimeout(fd int, cid uint32, port uint32, timeout time.Duration) error {
	if err := unix.SetNonblock(fd, true); err != nil {
		return fmt.Errorf("set nonblock: %w", err)
	}
	defer unix.SetNonblock(fd, false)

	addr := &unix.SockaddrVM{CID: cid, Port: port}
	err := unix.Connect(fd, addr)
	if err == nil {
		return nil
	}
	if !errors.Is(err, unix.EINPROGRESS) && !errors.Is(err, unix.EALREADY) && !errors.Is(err, unix.EWOULDBLOCK) {
		return fmt.Errorf("connect to %d:%d failed: %w", cid, port, err)
	}

	pollfds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLOUT}}
	ms := int(timeout.Milliseconds())
	if ms <= 0 {
		ms = 1
	}

	n, pollErr := unix.Poll(pollfds, ms)
	if pollErr != nil {
		return fmt.Errorf("poll while connecting to %d:%d failed: %w", cid, port, pollErr)
	}
	if n == 0 {
		return fmt.Errorf("connect to %d:%d failed: %w", cid, port, osOrTimeoutError(syscall.ETIMEDOUT))
	}

	soErr, getsockErr := unix.GetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_ERROR)
	if getsockErr != nil {
		return fmt.Errorf("getsockopt(SO_ERROR) for %d:%d failed: %w", cid, port, getsockErr)
	}
	if soErr != 0 {
		return fmt.Errorf("connect to %d:%d failed: %w", cid, port, syscall.Errno(soErr))
	}
	return nil
}

func applyReadWriteTimeout(fd int, timeout time.Duration) error {
	if err := applySockoptTimeout(fd, unix.SO_RCVTIMEO, timeout); err != nil {
		return err
	}
	return applySockoptTimeout(fd, unix.SO_SNDTIMEO, timeout)
}

func applySockoptTimeout(fd int, opt int, timeout time.Duration) error {
	if timeout < 0 {
		timeout = 0
	}
	tv := unix.NsecToTimeval(timeout.Nanoseconds())
	if err := unix.SetsockoptTimeval(fd, unix.SOL_SOCKET, opt, &tv); err != nil {
		return fmt.Errorf("set socket timeout: %w", err)
	}
	return nil
}

func osOrTimeoutError(err error) error {
	if errors.Is(err, syscall.EAGAIN) || errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.ETIMEDOUT) {
		return osTimeoutError{err: err}
	}
	return err
}

func timeUntil(t time.Time) time.Duration {
	if t.IsZero() {
		return 0
	}
	d := time.Until(t)
	if d < 0 {
		return 0
	}
	return d
}

type osTimeoutError struct {
	err error
}

func (e osTimeoutError) Error() string   { return e.err.Error() }
func (e osTimeoutError) Timeout() bool   { return true }
func (e osTimeoutError) Temporary() bool { return true }
