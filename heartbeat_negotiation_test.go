package stomp

import (
	"net"
	"testing"
	"time"

	"github.com/go-stomp/stomp/v3/frame"
	"github.com/stretchr/testify/require"
)

func heartbeatTestConn(t *testing.T, serverHeartbeat string, send, recv time.Duration) (*Conn, net.Conn) {
	t.Helper()
	client, server := net.Pipe()
	t.Cleanup(func() { _ = server.Close() })
	t.Cleanup(func() { _ = client.Close() })
	result := make(chan error, 1)
	go func() {
		_, err := frame.NewReader(server).Read()
		if err == nil {
			err = frame.NewWriter(server).Write(frame.New(frame.CONNECTED,
				frame.Version, "1.2", frame.HeartBeat, serverHeartbeat))
		}
		result <- err
	}()
	conn, err := Connect(client, ConnOpt.HeartBeat(send, recv), ConnOpt.HeartBeatError(0))
	require.NoError(t, err)
	require.NoError(t, <-result)
	t.Cleanup(func() { _ = conn.MustDisconnect() })
	return conn, server
}

func TestHeartBeatNegotiation(t *testing.T) {
	for _, tc := range []struct {
		name                string
		server              string
		send, recv          time.Duration
		wantRead, wantWrite time.Duration
	}{
		{"server disables both", "0,0", time.Second, time.Second, 0, 0},
		{"server disables sending", "0,1000", time.Second, time.Second, 0, time.Second},
		{"server disables receiving", "1000,0", time.Second, time.Second, time.Second, 0},
		{"client disables both", "1000,1000", 0, 0, 0, 0},
		{"client disables sending", "1000,1000", 0, time.Second, time.Second, 0},
		{"client disables receiving", "1000,1000", time.Second, 0, 0, time.Second},
		{"client slower", "1000,1000", 3 * time.Second, 2 * time.Second, 2 * time.Second, 3 * time.Second},
		{"server slower", "3000,2000", time.Second, time.Second, 3 * time.Second, 2 * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conn, _ := heartbeatTestConn(t, tc.server, tc.send, tc.recv)
			require.Equal(t, tc.wantRead, conn.readTimeout)
			require.Equal(t, tc.wantWrite, conn.writeTimeout)
		})
	}
}
