package stomp

import (
	"net"
	"testing"
	"time"

	"github.com/go-stomp/stomp/v3/frame"
	"github.com/stretchr/testify/require"
)

func receiptTestConn(t *testing.T, opts ...func(*Conn) error) (*Conn, net.Conn, *frame.Reader, *frame.Writer) {
	t.Helper()
	local, peer := net.Pipe()
	t.Cleanup(func() { _ = local.Close(); _ = peer.Close() })
	require.NoError(t, peer.SetDeadline(time.Now().Add(5*time.Second)))
	reader, writer := frame.NewReader(peer), frame.NewWriter(peer)
	done := make(chan error, 1)
	go func() {
		_, err := reader.Read()
		if err == nil {
			err = writer.Write(frame.New(frame.CONNECTED, frame.Version, "1.2", frame.HeartBeat, "0,0"))
		}
		done <- err
	}()
	c, err := Connect(local, opts...)
	require.NoError(t, err)
	require.NoError(t, <-done)
	t.Cleanup(func() { _ = c.MustDisconnect() })
	return c, peer, reader, writer
}

func TestLateReceiptDoesNotBlockConnection(t *testing.T) {
	c, _, reader, writer := receiptTestConn(t, ConnOpt.RcvReceiptTimeout(20*time.Millisecond))
	sent := make(chan error, 1)
	go func() { sent <- c.Send("/queue/a", "", nil, SendOpt.Receipt) }()
	first, err := reader.Read()
	require.NoError(t, err)
	require.ErrorIs(t, <-sent, ErrMsgReceiptTimeout)
	require.NoError(t, writer.Write(frame.New(frame.RECEIPT, frame.ReceiptId, first.Header.Get(frame.Receipt))))
	go func() { sent <- c.Send("/queue/b", "", nil, SendOpt.Receipt) }()
	second, err := reader.Read()
	require.NoError(t, err)
	require.Equal(t, "/queue/b", second.Header.Get(frame.Destination))
	require.NoError(t, writer.Write(frame.New(frame.RECEIPT, frame.ReceiptId, second.Header.Get(frame.Receipt))))
	require.NoError(t, <-sent)
}

func TestExpiredReceiptIsRemoved(t *testing.T) {
	c, _, reader, writer := receiptTestConn(t)
	response := make(chan *frame.Frame, 1)
	done := make(chan struct{})
	close(done)
	c.writeCh <- writeRequest{Frame: frame.New(frame.SEND, frame.Receipt, "expired"), C: response, Done: done}
	_, err := reader.Read()
	require.NoError(t, err)
	require.NoError(t, writer.Write(frame.New(frame.RECEIPT, frame.ReceiptId, "expired")))
	sent := make(chan error, 1)
	go func() { sent <- c.Send("/queue/a", "", nil, SendOpt.Receipt) }()
	f, err := reader.Read()
	require.NoError(t, err)
	require.NoError(t, writer.Write(frame.New(frame.RECEIPT, frame.ReceiptId, f.Header.Get(frame.Receipt))))
	require.NoError(t, <-sent)
	require.Empty(t, response)
}

func TestConnectionCloseCompletesPendingReceipt(t *testing.T) {
	c, peer, reader, _ := receiptTestConn(t, ConnOpt.RcvReceiptTimeout(0))
	sent := make(chan error, 1)
	go func() { sent <- c.Send("/queue/a", "", nil, SendOpt.Receipt) }()
	_, err := reader.Read()
	require.NoError(t, err)
	require.NoError(t, peer.Close())
	select {
	case err := <-sent:
		require.Error(t, err)
	case <-time.After(time.Second):
		t.Fatal("receipt wait did not finish")
	}
}

func TestMustDisconnectCompletesPendingReceipt(t *testing.T) {
	c, _, reader, _ := receiptTestConn(t, ConnOpt.RcvReceiptTimeout(0))
	sent := make(chan error, 1)
	go func() { sent <- c.Send("/queue/a", "", nil, SendOpt.Receipt) }()
	_, err := reader.Read()
	require.NoError(t, err)
	disconnected := make(chan error, 1)
	go func() { disconnected <- c.MustDisconnect() }()
	select {
	case err := <-sent:
		require.ErrorIs(t, err, ErrClosedUnexpectedly)
	case <-time.After(time.Second):
		t.Fatal("receipt wait did not finish")
	}
	require.NoError(t, <-disconnected)
}

func TestDeliveredResponseWinsOverClose(t *testing.T) {
	closed := make(chan struct{})
	close(closed)
	// a random select between two ready channels would fail this quickly
	for i := 0; i < 100; i++ {
		responses := make(chan *frame.Frame, 1)
		responses <- frame.New(frame.ERROR, frame.Message, "broker said no")
		err := readReceiptWithTimeout(responses, 0, ErrMsgReceiptTimeout, closed)
		require.EqualError(t, err, "broker said no")

		responses = make(chan *frame.Frame, 1)
		responses <- frame.New(frame.RECEIPT, frame.ReceiptId, "1")
		require.NoError(t, readReceiptWithTimeout(responses, 0, ErrMsgReceiptTimeout, closed))
	}
	require.ErrorIs(t, readReceiptWithTimeout(make(chan *frame.Frame, 1), 0, ErrMsgReceiptTimeout, closed), ErrClosedUnexpectedly)
}

func TestUnsubscribeCompletesCallerReceiptAndSubscription(t *testing.T) {
	for _, sameID := range []bool{false, true} {
		name := "different IDs"
		if sameID {
			name = "same ID"
		}
		t.Run(name, func(t *testing.T) {
			c, _, reader, writer := receiptTestConn(t, ConnOpt.UnsubscribeReceiptTimeout(0))
			sub, err := c.Subscribe("/queue/a", AckAuto)
			require.NoError(t, err)
			_, err = reader.Read()
			require.NoError(t, err)
			receipt := "unsubscribe-receipt"
			if sameID {
				receipt = sub.Id()
			}
			done := make(chan error, 1)
			go func() { done <- sub.Unsubscribe(SubscribeOpt.Header(frame.Receipt, receipt)) }()
			f, err := reader.Read()
			require.NoError(t, err)
			require.Equal(t, receipt, f.Header.Get(frame.Receipt))
			require.NoError(t, writer.Write(frame.New(frame.RECEIPT, frame.ReceiptId, receipt)))
			select {
			case err := <-done:
				require.NoError(t, err)
			case <-time.After(time.Second):
				t.Fatal("unsubscribe did not finish after its receipt")
			}
			_, open := <-sub.C
			require.False(t, open)
			require.False(t, sub.Active())
			require.NoError(t, c.Send("/queue/b", "", nil))
			f, err = reader.Read()
			require.NoError(t, err)
			require.Equal(t, "/queue/b", f.Header.Get(frame.Destination))
		})
	}
}
