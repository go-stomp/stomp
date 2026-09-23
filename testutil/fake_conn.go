package testutil

import (
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

type FakeAddr struct {
	Value string
}

func (addr *FakeAddr) Network() string {
	return "fake"
}

func (addr *FakeAddr) String() string {
	return addr.Value
}

// FakeConn is a fake connection used for testing. It implements
// the net.Conn interface and is useful for simulating I/O between
// STOMP clients and a STOMP server.
type FakeConn struct {
	T            testing.TB
	writer       io.WriteCloser
	reader       io.ReadCloser
	localAddr    net.Addr
	remoteAddr   net.Addr
	readDeadline time.Time
}

var (
	ErrClosing   = errors.New("use of closed network connection")
	ErrIOTimeout = errors.New("io timeout")
)

// NewFakeConn returns a pair of fake connections suitable for
// testing.
func NewFakeConn(t testing.TB) (client *FakeConn, server *FakeConn) {
	clientReader, serverWriter := io.Pipe()
	serverReader, clientWriter := io.Pipe()
	clientAddr := &FakeAddr{Value: "the-client:123"}
	serverAddr := &FakeAddr{Value: "the-server:456"}

	clientConn := &FakeConn{
		T:          t,
		reader:     clientReader,
		writer:     clientWriter,
		localAddr:  clientAddr,
		remoteAddr: serverAddr,
	}

	serverConn := &FakeConn{
		T:          t,
		reader:     serverReader,
		writer:     serverWriter,
		localAddr:  serverAddr,
		remoteAddr: clientAddr,
	}

	return clientConn, serverConn
}

func (fc *FakeConn) Read(p []byte) (n int, err error) {
	if !fc.readDeadline.IsZero() {
		t := time.Until(fc.readDeadline)
		if t.Seconds() > 0 {
			time.Sleep(t)
		}
		return 0, ErrIOTimeout
	}
	n, err = fc.reader.Read(p)
	return
}

func (fc *FakeConn) Write(p []byte) (n int, err error) {
	return fc.writer.Write(p)
}

func (fc *FakeConn) Close() error {
	err1 := fc.reader.Close()
	err2 := fc.writer.Close()

	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}
	return nil
}

func (fc *FakeConn) LocalAddr() net.Addr {
	return fc.localAddr
}

func (fc *FakeConn) RemoteAddr() net.Addr {
	return fc.remoteAddr
}

func (fc *FakeConn) SetLocalAddr(addr net.Addr) {
	fc.localAddr = addr
}

func (fc *FakeConn) SetRemoteAddr(addr net.Addr) {
	fc.remoteAddr = addr
}

func (fc *FakeConn) SetDeadline(t time.Time) error {
	panic("not implemented")
}

func (fc *FakeConn) SetReadDeadline(t time.Time) error {
	fc.readDeadline = t
	return nil
}

func (fc *FakeConn) SetWriteDeadline(t time.Time) error {
	panic("not implemented")
}
