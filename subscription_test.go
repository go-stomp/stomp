package stomp

import (
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
	defer fc1.Close()
	defer fc2.Close()

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
	defer fc1.Close()
	defer fc2.Close()

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

// serverOperation is a function that performs a server operation that either reads or writes a frame and returns it.
type serverOperation func(t testing.TB, reader *frame.Reader, writer *frame.Writer, previousFrames []*frame.Frame) (*frame.Frame, error)

func runFakeConn(t testing.TB, operations ...serverOperation) (*sync.WaitGroup, *testutil.FakeConn, *testutil.FakeConn) {
	client, server := testutil.NewFakeConn(t)

	wg := &sync.WaitGroup{}
	go func() {
		wg.Add(1)
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
