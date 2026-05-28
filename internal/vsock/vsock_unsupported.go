//go:build !linux

package vsock

import (
	"errors"
	"io"
	"net"
	"time"
)

const HostCID uint32 = 2

var errVsockUnsupported = errors.New("vsock is only supported on linux")

type Addr struct {
	CID  uint32
	Port uint32
}

func (a Addr) Network() string { return "vsock" }
func (a Addr) String() string  { return "unsupported" }

type Conn struct{}

func Dial(cid uint32, port uint32, timeout time.Duration) (*Conn, error) {
	return nil, errVsockUnsupported
}

func LocalCID() (uint32, error) {
	return 0, errVsockUnsupported
}

func (c *Conn) Read(_ []byte) (int, error)         { return 0, io.EOF }
func (c *Conn) Write(p []byte) (int, error)        { return len(p), nil }
func (c *Conn) Close() error                       { return nil }
func (c *Conn) LocalAddr() net.Addr                { return Addr{} }
func (c *Conn) RemoteAddr() net.Addr               { return Addr{} }
func (c *Conn) SetDeadline(_ time.Time) error      { return nil }
func (c *Conn) SetReadDeadline(_ time.Time) error  { return nil }
func (c *Conn) SetWriteDeadline(_ time.Time) error { return nil }
