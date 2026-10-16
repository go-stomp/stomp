package stomp

import (
	"context"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/go-stomp/stomp/v3/frame"
	"github.com/stretchr/testify/require"
)

type handshakeStream struct{ io.ReadWriteCloser }

type observedHandshakeConn struct {
	net.Conn
	started chan struct{}
	once    sync.Once
}

func (c *observedHandshakeConn) Write(p []byte) (int, error) {
	c.once.Do(func() { close(c.started) })
	return c.Conn.Write(p)
}

func TestConnectContextCancellation(t *testing.T) {
	for _, phase := range []string{"write", "read"} {
		for _, wrapped := range []bool{false, true} {
			for _, deadline := range []bool{false, true} {
				t.Run(phase+map[bool]string{false: "/socket", true: "/stream"}[wrapped]+map[bool]string{false: "/cancel", true: "/deadline"}[deadline], func(t *testing.T) {
					local, peer := net.Pipe()
					defer func() { _ = local.Close() }()
					defer func() { _ = peer.Close() }()
					observed := &observedHandshakeConn{Conn: local, started: make(chan struct{})}
					var conn io.ReadWriteCloser = observed
					if wrapped {
						conn = handshakeStream{observed}
					}
					ctx, cancel := context.WithCancel(context.Background())
					if deadline {
						cancel()
						ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
					}
					defer cancel()
					read := make(chan error, 1)
					if phase == "read" {
						go func() { _, err := frame.NewReader(peer).Read(); read <- err }()
					}
					result := make(chan error, 1)
					go func() { _, err := ConnectWithContext(ctx, conn); result <- err }()
					if phase == "read" {
						require.NoError(t, <-read)
					}
					if phase == "write" {
						<-observed.started
					}
					if !deadline {
						cancel()
					}
					select {
					case err := <-result:
						if deadline {
							require.ErrorIs(t, err, context.DeadlineExceeded)
						} else {
							require.ErrorIs(t, err, context.Canceled)
						}
					case <-time.After(time.Second):
						t.Fatal("handshake did not stop")
					}
				})
			}
		}
	}
}

func TestDialContextAlreadyCanceled(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = listener.Close() }()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c, err := DialWithContext(ctx, "tcp", listener.Addr().String())
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, c)
}

func TestConnectContextEndsWithHandshake(t *testing.T) {
	local, peer := net.Pipe()
	defer func() { _ = local.Close() }()
	defer func() { _ = peer.Close() }()
	require.NoError(t, peer.SetDeadline(time.Now().Add(time.Second)))
	reader, writer := frame.NewReader(peer), frame.NewWriter(peer)
	done := make(chan error, 1)
	go func() {
		_, err := reader.Read()
		if err == nil {
			err = writer.Write(frame.New(frame.CONNECTED, frame.Version, "1.2", frame.HeartBeat, "0,0"))
		}
		done <- err
	}()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c, err := ConnectWithContext(ctx, local)
	require.NoError(t, err)
	defer func() { _ = c.MustDisconnect() }()
	require.NoError(t, <-done)
	cancel()
	require.NoError(t, c.Send("/queue/a", "", []byte("body")))
	f, err := reader.Read()
	require.NoError(t, err)
	require.Equal(t, "body", string(f.Body))
}
