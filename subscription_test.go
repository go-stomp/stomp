package stomp

import (
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/go-stomp/stomp/v3/frame"
	"github.com/go-stomp/stomp/v3/testutil"
	"github.com/stretchr/testify/require"
)

func TestStompSuccessfulUnsubscribeWithReceiptTimeout(t *testing.T) {
	wg, fc1, fc2 := runFakeConn(t,
		assertConnectFrame,
		sendConnectedFrame,
		assertSubscribeFrame,
		assertUnsubscribeFrame,
	)
	defer func() { _ = fc1.Close() }()
	defer func() { _ = fc2.Close() }()

	client, err := Connect(fc1, ConnOpt.UnsubscribeReceiptTimeout(1*time.Second))
	require.NoError(t, err)
	require.NotNil(t, client)

	sub, err := client.Subscribe("/queue/test", AckAuto)
	require.NoError(t, err)
	require.NotNil(t, sub)

	err = sub.Unsubscribe()
	require.ErrorIs(t, err, ErrUnsubscribeReceiptTimeout)
	wg.Wait()
}

func TestStompSuccessfulUnsubscribeNoTimeout(t *testing.T) {
	wg, fc1, fc2 := runFakeConn(t,
		assertConnectFrame,
		sendConnectedFrame,
		assertSubscribeFrame,
		assertUnsubscribeFrame,
		sendReceiptFrame(3),
	)
	defer func() { _ = fc1.Close() }()
	defer func() { _ = fc2.Close() }()

	client, err := Connect(fc1)
	require.NoError(t, err)
	require.NotNil(t, client)

	sub, err := client.Subscribe("/queue/test", AckAuto)
	require.NoError(t, err)
	require.NotNil(t, sub)

	err = sub.Unsubscribe()
	require.NoError(t, err)
	wg.Wait()
}

func TestStompUnsubscribeWithCallerReceipt(t *testing.T) {
	wg, fc1, fc2 := runFakeConn(t,
		assertConnectFrame,
		sendConnectedFrame,
		assertSubscribeFrame,
		func(t testing.TB, reader *frame.Reader, writer *frame.Writer, _ []*frame.Frame) (*frame.Frame, error) {
			f, err := assertUnsubscribeFrame(t, reader, writer, nil)
			require.Equal(t, "r1", f.Header.Get(frame.Receipt))
			return f, err
		},
		func(t testing.TB, _ *frame.Reader, writer *frame.Writer, _ []*frame.Frame) (*frame.Frame, error) {
			f := frame.New(frame.RECEIPT, frame.ReceiptId, "r1")
			return f, writer.Write(f)
		},
	)
	defer func() { _ = fc1.Close() }()
	defer func() { _ = fc2.Close() }()

	client, err := Connect(fc1)
	require.NoError(t, err)

	sub, err := client.Subscribe("/queue/test", AckAuto)
	require.NoError(t, err)

	done := make(chan error, 1)
	go func() {
		done <- sub.Unsubscribe(SubscribeOpt.Header(frame.Receipt, "r1"))
	}()

	select {
	case err = <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Unsubscribe did not complete after the RECEIPT for the caller's receipt id")
	}
	require.False(t, sub.Active())
	wg.Wait()
}

// serverOperation is a function that performs a server operation that either reads or writes a frame and returns it.
type serverOperation func(t testing.TB, reader *frame.Reader, writer *frame.Writer, previousFrames []*frame.Frame) (*frame.Frame, error)

func runFakeConn(t testing.TB, operations ...serverOperation) (*sync.WaitGroup, *testutil.FakeConn, *testutil.FakeConn) {
	client, server := testutil.NewFakeConn(t)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		reader := frame.NewReader(server)
		writer := frame.NewWriter(server)

		frames := make([]*frame.Frame, 0)
		for _, operation := range operations {
			frame, err := operation(t, reader, writer, frames)
			frames = append(frames, frame)
			if err != nil {
				t.Fatalf("error in server operation: %v", err)
				return
			}
		}
	}()

	return wg, client, server
}

func assertConnectFrame(t testing.TB, reader *frame.Reader, _ *frame.Writer, _ []*frame.Frame) (*frame.Frame, error) {
	f, err := reader.Read()
	require.NoError(t, err)
	require.Equal(t, frame.CONNECT, f.Command)
	return f, err
}

func sendConnectedFrame(t testing.TB, _ *frame.Reader, writer *frame.Writer, _ []*frame.Frame) (*frame.Frame, error) {
	f := frame.New(frame.CONNECTED)
	err := writer.Write(f)
	require.NoError(t, err)
	return f, err
}

func assertSubscribeFrame(t testing.TB, reader *frame.Reader, _ *frame.Writer, _ []*frame.Frame) (*frame.Frame, error) {
	f, err := reader.Read()
	require.NoError(t, err)
	require.Equal(t, frame.SUBSCRIBE, f.Command)
	return f, err
}

func assertUnsubscribeFrame(t testing.TB, reader *frame.Reader, _ *frame.Writer, _ []*frame.Frame) (*frame.Frame, error) {
	f, err := reader.Read()
	require.NoError(t, err)
	require.Equal(t, frame.UNSUBSCRIBE, f.Command)
	return f, err
}

// sendReceiptFrame returns a server operation that writes a RECEIPT frame to the writer based on the id of the previous frame
func sendReceiptFrame(frameId int) serverOperation {
	return func(t testing.TB, _ *frame.Reader, writer *frame.Writer, previousFrames []*frame.Frame) (*frame.Frame, error) {
		f := frame.New(frame.RECEIPT)
		previousFrame := previousFrames[frameId]
		require.NotNil(t, previousFrame)
		f.Header.Set(frame.ReceiptId, previousFrame.Header.Get(frame.Id))
		err := writer.Write(f)
		require.NoError(t, err)
		return f, err
	}
}

// floodServer is a fake broker that answers the first SUBSCRIBE with n
// MESSAGEs and confirms UNSUBSCRIBE with a RECEIPT. DISCONNECT is never
// confirmed. Any other frame it receives is passed on to the returned channel.
func floodServer(fc *testutil.FakeConn, n int) <-chan *frame.Frame {
	received := make(chan *frame.Frame, 100)
	go func() {
		defer close(received)
		reader := frame.NewReader(fc)
		writer := frame.NewWriter(fc)
		f, err := reader.Read()
		if err != nil || f.Command != frame.CONNECT {
			return
		}
		if writer.Write(frame.New(frame.CONNECTED, frame.Version, "1.2")) != nil {
			return
		}

		// the client may stop reading, so writes must not hold up reading
		out := make(chan *frame.Frame, n+100)
		defer close(out)
		go func() {
			for f := range out {
				if writer.Write(f) != nil {
					return
				}
			}
		}()

		subscribed := false
		for {
			f, err := reader.Read()
			if err != nil {
				return
			}
			if f == nil {
				continue
			}
			switch {
			case f.Command == frame.SUBSCRIBE && !subscribed:
				subscribed = true
				id := f.Header.Get(frame.Id)
				for i := 0; i < n; i++ {
					out <- frame.New(frame.MESSAGE,
						frame.Subscription, id,
						frame.Destination, "/queue/test",
						frame.MessageId, strconv.Itoa(i))
				}
			case f.Command == frame.UNSUBSCRIBE:
				out <- frame.New(frame.RECEIPT, frame.ReceiptId, f.Header.Get(frame.Receipt))
			case f.Command != frame.DISCONNECT:
				received <- f
			}
		}
	}()
	return received
}

// Unsubscribe called by the goroutine reading C, while more messages are in
// flight than C can hold, used to block readLoop on the full C and
// processLoop behind it, so the UNSUBSCRIBE never even reached the wire.
func TestUnsubscribeFromConsumerDoesNotDeadlock(t *testing.T) {
	fc1, fc2 := testutil.NewFakeConn(t)
	defer func() { _ = fc1.Close() }()
	floodServer(fc2, 100)

	client, err := Connect(fc1, ConnOpt.UnsubscribeReceiptTimeout(0))
	require.NoError(t, err)
	sub, err := client.Subscribe("/queue/test", AckAuto)
	require.NoError(t, err)

	<-sub.C
	// let the in-flight messages fill C
	time.Sleep(100 * time.Millisecond)

	done := make(chan error, 1)
	go func() { done <- sub.Unsubscribe() }()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Unsubscribe blocked")
	}
}
