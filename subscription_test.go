package stomp

import (
	"github.com/go-stomp/stomp/v3/frame"
	"github.com/go-stomp/stomp/v3/testutil"
	"sync"
	"time"

	. "gopkg.in/check.v1"
)

func (s *StompSuite) Test_successful_unsubscribe_with_receipt_timeout(c *C) {
	resetId()

	wg, fc1, fc2 := runFakeConn(c,
		assertConnectFrame,
		sendConnectedFrame,
		assertSubscribeFrame,
		assertUnsubscribeFrame,
	)
	defer fc1.Close()
	defer fc2.Close()

	client, err := Connect(fc1, ConnOpt.UnsubscribeReceiptTimeout(1*time.Second))
	c.Assert(err, IsNil)
	c.Assert(client, NotNil)

	sub, err := client.Subscribe("/queue/test", AckAuto)
	c.Assert(err, IsNil)
	c.Assert(sub, NotNil)

	err = sub.Unsubscribe()
	c.Assert(err, Equals, &ErrUnsubscribeReceiptTimeout)
	wg.Wait()
}

func (s *StompSuite) Test_successful_unsubscribe_no_timeout(c *C) {
	resetId()

	wg, fc1, fc2 := runFakeConn(c,
		assertConnectFrame,
		sendConnectedFrame,
		assertSubscribeFrame,
		assertUnsubscribeFrame,
		sendReceiptFrame(3),
	)
	defer fc1.Close()
	defer fc2.Close()

	client, err := Connect(fc1)
	c.Assert(err, IsNil)
	c.Assert(client, NotNil)

	sub, err := client.Subscribe("/queue/test", AckAuto)
	c.Assert(err, IsNil)
	c.Assert(sub, NotNil)

	err = sub.Unsubscribe()
	c.Assert(err, IsNil)
	wg.Wait()
}

// serverOperation is a function that performs a server operation that either reads or writes a frame and returns it.
type serverOperation func(c *C, reader *frame.Reader, writer *frame.Writer, previousFrames []*frame.Frame) (*frame.Frame, error)

func runFakeConn(c *C, operations ...serverOperation) (*sync.WaitGroup, *testutil.FakeConn, *testutil.FakeConn) {
	client, server := testutil.NewFakeConn(c)

	wg := &sync.WaitGroup{}
	go func() {
		wg.Add(1)
		defer wg.Done()
		reader := frame.NewReader(server)
		writer := frame.NewWriter(server)

		frames := make([]*frame.Frame, 0)
		for _, operation := range operations {
			frame, err := operation(c, reader, writer, frames)
			frames = append(frames, frame)
			if err != nil {
				c.Errorf("error in server operation: %v", err)
				return
			}
		}
	}()

	return wg, client, server
}

func assertConnectFrame(c *C, reader *frame.Reader, _ *frame.Writer, _ []*frame.Frame) (*frame.Frame, error) {
	f, err := reader.Read()
	c.Assert(err, IsNil)
	c.Assert(f.Command, Equals, frame.CONNECT)
	return f, err
}

func sendConnectedFrame(c *C, _ *frame.Reader, writer *frame.Writer, _ []*frame.Frame) (*frame.Frame, error) {
	f := frame.New(frame.CONNECTED)
	err := writer.Write(f)
	c.Assert(err, IsNil)
	return f, err
}

func assertSubscribeFrame(c *C, reader *frame.Reader, _ *frame.Writer, _ []*frame.Frame) (*frame.Frame, error) {
	f, err := reader.Read()
	c.Assert(err, IsNil)
	c.Assert(f.Command, Equals, frame.SUBSCRIBE)
	return f, err
}

func assertUnsubscribeFrame(c *C, reader *frame.Reader, _ *frame.Writer, _ []*frame.Frame) (*frame.Frame, error) {
	f, err := reader.Read()
	c.Assert(err, IsNil)
	c.Assert(f.Command, Equals, frame.UNSUBSCRIBE)
	return f, err
}

// sendReceiptFrame returns a server operation that writes a RECEIPT frame to the writer based on the id of the previous frame
func sendReceiptFrame(frameId int) serverOperation {
	return func(c *C, _ *frame.Reader, writer *frame.Writer, previousFrames []*frame.Frame) (*frame.Frame, error) {
		f := frame.New(frame.RECEIPT)
		previousFrame := previousFrames[frameId]
		c.Assert(previousFrame, NotNil)
		f.Header.Set(frame.ReceiptId, previousFrame.Header.Get(frame.Id))
		err := writer.Write(f)
		c.Assert(err, IsNil)
		return f, err
	}
}
