package stomp

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/go-stomp/stomp/v3/frame"
	"github.com/go-stomp/stomp/v3/testutil"
	"github.com/stretchr/testify/require"

	"github.com/golang/mock/gomock"
)

type fakeReaderWriter struct {
	reader *frame.Reader
	writer *frame.Writer
	conn   io.ReadWriteCloser
}

func (rw *fakeReaderWriter) Read() (*frame.Frame, error) {
	return rw.reader.Read()
}

func (rw *fakeReaderWriter) Write(f *frame.Frame) error {
	return rw.writer.Write(f)
}

func (rw *fakeReaderWriter) Close() error {
	return rw.conn.Close()
}

func TestStompConnOptionSetLogger(t *testing.T) {
	fc1, fc2 := testutil.NewFakeConn(t)
	go func() {

		defer func() {
			fc2.Close()
			fc1.Close()
		}()

		reader := frame.NewReader(fc2)
		writer := frame.NewWriter(fc2)
		f1, err := reader.Read()
		require.NoError(t, err)
		require.Equal(t, "CONNECT", f1.Command)
		f2 := frame.New("CONNECTED")
		err = writer.Write(f2)
		require.NoError(t, err)
	}()

	ctrl := gomock.NewController(t)
	mockLogger := testutil.NewMockLogger(ctrl)

	conn, err := Connect(fc1, ConnOpt.Logger(mockLogger))
	require.NoError(t, err)
	require.NotNil(t, conn)

	require.Equal(t, mockLogger, conn.log)
}

func TestStompUnsuccessfulConnect(t *testing.T) {
	fc1, fc2 := testutil.NewFakeConn(t)
	stop := make(chan struct{})

	go func() {
		defer func() {
			fc2.Close()
			close(stop)
		}()

		reader := frame.NewReader(fc2)
		writer := frame.NewWriter(fc2)
		f1, err := reader.Read()
		require.NoError(t, err)
		require.Equal(t, "CONNECT", f1.Command)
		f2 := frame.New("ERROR", "message", "auth-failed")
		err = writer.Write(f2)
		require.NoError(t, err)
	}()

	conn, err := Connect(fc1)
	require.Nil(t, conn)
	require.Equal(t, "auth-failed", err.Error())
}

func TestSuccessfulConnectAndDisconnect(t *testing.T) {
	testcases := []struct {
		Options           []func(*Conn) error
		NegotiatedVersion string
		ExpectedVersion   Version
		ExpectedSession   string
		ExpectedHost      string
		ExpectedServer    string
	}{
		{
			Options:         []func(*Conn) error{ConnOpt.Host("the-server")},
			ExpectedVersion: "1.0",
			ExpectedSession: "",
			ExpectedHost:    "the-server",
			ExpectedServer:  "some-server/1.1",
		},
		{
			Options:           []func(*Conn) error{},
			NegotiatedVersion: "1.1",
			ExpectedVersion:   "1.1",
			ExpectedSession:   "the-session",
			ExpectedHost:      "the-server",
		},
		{
			Options:           []func(*Conn) error{ConnOpt.Host("xxx")},
			NegotiatedVersion: "1.2",
			ExpectedVersion:   "1.2",
			ExpectedSession:   "the-session",
			ExpectedHost:      "xxx",
		},
	}

	for _, tc := range testcases {
		fc1, fc2 := testutil.NewFakeConn(t)
		stop := make(chan struct{})

		go func() {
			defer func() {
				fc2.Close()
				close(stop)
			}()
			reader := frame.NewReader(fc2)
			writer := frame.NewWriter(fc2)

			f1, err := reader.Read()
			require.NoError(t, err)
			require.Equal(t, "CONNECT", f1.Command)
			host, _ := f1.Header.Contains("host")
			require.Equal(t, tc.ExpectedHost, host)
			connectedFrame := frame.New("CONNECTED")
			if tc.NegotiatedVersion != "" {
				connectedFrame.Header.Add("version", tc.NegotiatedVersion)
			}
			if tc.ExpectedSession != "" {
				connectedFrame.Header.Add("session", tc.ExpectedSession)
			}
			if tc.ExpectedServer != "" {
				connectedFrame.Header.Add("server", tc.ExpectedServer)
			}
			err = writer.Write(connectedFrame)
			require.NoError(t, err)

			f2, err := reader.Read()
			require.NoError(t, err)
			require.Equal(t, "DISCONNECT", f2.Command)
			receipt, _ := f2.Header.Contains("receipt")
			require.Equal(t, "1", receipt)

			err = writer.Write(frame.New("RECEIPT", frame.ReceiptId, "1"))
			require.NoError(t, err)

		}()

		client, err := Connect(fc1, tc.Options...)
		require.NoError(t, err)
		require.NotNil(t, client)
		require.Equal(t, tc.ExpectedVersion, client.Version())
		require.Equal(t, tc.ExpectedSession, client.Session())
		require.Equal(t, tc.ExpectedServer, client.Server())

		err = client.Disconnect()
		require.NoError(t, err)

		<-stop
	}
}

func TestStompSuccessfulConnectGetHeaders(t *testing.T) {
	var respHeaders *frame.Header

	testcases := []struct {
		Options []func(*Conn) error
		Headers map[string]string
	}{
		{
			Options: []func(*Conn) error{ConnOpt.ResponseHeaders(func(f *frame.Header) { respHeaders = f })},
			Headers: map[string]string{"custom-header": "test", "foo": "bar"},
		},
	}

	for _, tc := range testcases {
		fc1, fc2 := testutil.NewFakeConn(t)
		stop := make(chan struct{})

		go func() {
			defer func() {
				fc2.Close()
				close(stop)
			}()
			reader := frame.NewReader(fc2)
			writer := frame.NewWriter(fc2)

			f1, err := reader.Read()
			require.NoError(t, err)
			require.Equal(t, "CONNECT", f1.Command)
			connectedFrame := frame.New("CONNECTED")
			for key, value := range tc.Headers {
				connectedFrame.Header.Add(key, value)
			}
			err = writer.Write(connectedFrame)
			require.NoError(t, err)

			f2, err := reader.Read()
			require.NoError(t, err)
			require.Equal(t, "DISCONNECT", f2.Command)
			receipt, _ := f2.Header.Contains("receipt")
			require.Equal(t, "1", receipt)

			err = writer.Write(frame.New("RECEIPT", frame.ReceiptId, "1"))
			require.NoError(t, err)

		}()

		client, err := Connect(fc1, tc.Options...)
		require.NoError(t, err)
		require.NotNil(t, client)
		require.NotNil(t, respHeaders)
		for key, value := range tc.Headers {
			require.Equal(t, value, respHeaders.Get(key))
		}
		err = client.Disconnect()
		require.NoError(t, err)

		<-stop
	}
}

func TestStompSuccessfulConnectWithNonstandardHeader(t *testing.T) {
	fc1, fc2 := testutil.NewFakeConn(t)
	stop := make(chan struct{})

	go func() {
		defer func() {
			fc2.Close()
			close(stop)
		}()
		reader := frame.NewReader(fc2)
		writer := frame.NewWriter(fc2)

		f1, err := reader.Read()
		require.NoError(t, err)
		require.Equal(t, "CONNECT", f1.Command)
		require.Equal(t, "guest", f1.Header.Get("login"))
		require.Equal(t, "guest", f1.Header.Get("passcode"))
		require.Equal(t, "/", f1.Header.Get("host"))
		require.Equal(t, "50", f1.Header.Get("x-max-length"))
		connectedFrame := frame.New("CONNECTED")
		connectedFrame.Header.Add("session", "session-0voRHrG-VbBedx1Gwwb62Q")
		connectedFrame.Header.Add("heart-beat", "0,0")
		connectedFrame.Header.Add("server", "RabbitMQ/3.2.1")
		connectedFrame.Header.Add("version", "1.0")
		err = writer.Write(connectedFrame)
		require.NoError(t, err)

		f2, err := reader.Read()
		require.NoError(t, err)
		require.Equal(t, "DISCONNECT", f2.Command)
		receipt, _ := f2.Header.Contains("receipt")
		require.Equal(t, "1", receipt)

		err = writer.Write(frame.New("RECEIPT", frame.ReceiptId, "1"))
		require.NoError(t, err)
	}()

	client, err := Connect(fc1,
		ConnOpt.Login("guest", "guest"),
		ConnOpt.Host("/"),
		ConnOpt.Header("x-max-length", "50"))
	require.NoError(t, err)
	require.NotNil(t, client)
	require.Equal(t, V10, client.Version())
	require.Equal(t, "session-0voRHrG-VbBedx1Gwwb62Q", client.Session())
	require.Equal(t, "RabbitMQ/3.2.1", client.Server())

	err = client.Disconnect()
	require.NotNil(t, client)

	<-stop
}

func TestStompConnectNotPanicOnEmptyResponse(t *testing.T) {
	fc1, fc2 := testutil.NewFakeConn(t)
	stop := make(chan struct{})

	go func() {
		defer func() {
			fc2.Close()
			close(stop)
		}()
		reader := frame.NewReader(fc2)
		_, err := reader.Read()
		require.NoError(t, err)
		_, err = fc2.Write([]byte("\n"))
		require.NoError(t, err)
	}()

	client, err := Connect(fc1, ConnOpt.Host("the_server"))
	require.Error(t, err)
	require.Nil(t, client)

	fc1.Close()
	<-stop
}

func TestSuccessfulDisconnectWithReceiptTimeout(t *testing.T) {
	fc1, fc2 := testutil.NewFakeConn(t)

	defer func() {
		fc2.Close()
	}()

	go func() {
		reader := frame.NewReader(fc2)
		writer := frame.NewWriter(fc2)

		f1, err := reader.Read()
		require.NoError(t, err)
		require.Equal(t, "CONNECT", f1.Command)
		connectedFrame := frame.New("CONNECTED")
		err = writer.Write(connectedFrame)
		require.NoError(t, err)
	}()

	client, err := Connect(fc1, ConnOpt.DisconnectReceiptTimeout(1*time.Nanosecond))
	require.NoError(t, err)
	require.NotNil(t, client)

	err = client.Disconnect()
	require.ErrorIs(t, err, ErrDisconnectReceiptTimeout)
	require.True(t, client.closed)
}

// Sets up a connection for testing
func connectHelper(t testing.TB, version Version) (*Conn, *fakeReaderWriter) {
	fc1, fc2 := testutil.NewFakeConn(t)
	stop := make(chan struct{})

	reader := frame.NewReader(fc2)
	writer := frame.NewWriter(fc2)

	go func() {
		f1, err := reader.Read()
		require.NoError(t, err)
		require.Equal(t, "CONNECT", f1.Command)
		f2 := frame.New("CONNECTED", "version", version.String())
		err = writer.Write(f2)
		require.NoError(t, err)
		close(stop)
	}()

	conn, err := Connect(fc1)
	require.NoError(t, err)
	require.NotNil(t, conn)
	<-stop
	return conn, &fakeReaderWriter{
		reader: reader,
		writer: writer,
		conn:   fc2,
	}
}

func TestStompSubscribe(t *testing.T) {
	ackModes := []AckMode{AckAuto, AckClient, AckClientIndividual}
	versions := []Version{V10, V11, V12}

	for _, ackMode := range ackModes {
		for _, version := range versions {
			subscribeHelper(t, ackMode, version)
			subscribeHelper(t, ackMode, version,
				SubscribeOpt.Header("id", "client-1"),
				SubscribeOpt.Header("custom", "true"))
		}
	}
}

func subscribeHelper(t testing.TB, ackMode AckMode, version Version, opts ...func(*frame.Frame) error) {
	conn, rw := connectHelper(t, version)
	stop := make(chan struct{})

	go func() {
		defer func() {
			rw.Close()
			close(stop)
		}()

		f3, err := rw.Read()
		require.NoError(t, err)
		require.Equal(t, "SUBSCRIBE", f3.Command)

		id, ok := f3.Header.Contains("id")
		require.True(t, ok)

		destination := f3.Header.Get("destination")
		require.Equal(t, "/queue/test-1", destination)
		ack := f3.Header.Get("ack")
		require.Equal(t, ackMode.String(), ack)

		for i := 1; i <= 5; i++ {
			messageId := fmt.Sprintf("message-%d", i)
			bodyText := fmt.Sprintf("Message body %d", i)
			f4 := frame.New("MESSAGE",
				frame.Subscription, id,
				frame.MessageId, messageId,
				frame.Destination, destination)
			if version == V12 && ackMode.ShouldAck() {
				f4.Header.Add(frame.Ack, messageId)
			}
			f4.Body = []byte(bodyText)
			err = rw.Write(f4)
			require.NoError(t, err)

			if ackMode.ShouldAck() {
				f5, _ := rw.Read()
				require.Equal(t, "ACK", f5.Command)
				if version == V12 {
					require.Equal(t, messageId, f5.Header.Get(frame.Id))
				} else {
					require.Equal(t, id, f5.Header.Get("subscription"))
					require.Equal(t, messageId, f5.Header.Get("message-id"))
				}
			}
		}

		f6, _ := rw.Read()
		require.Equal(t, "UNSUBSCRIBE", f6.Command)
		require.NotEqual(t, "", f6.Header.Get(frame.Receipt))
		require.Equal(t, id, f6.Header.Get(frame.Id))
		err = rw.Write(frame.New(frame.RECEIPT, frame.ReceiptId, f6.Header.Get(frame.Receipt)))
		require.NoError(t, err)

		f7, _ := rw.Read()
		require.Equal(t, "DISCONNECT", f7.Command)
		err = rw.Write(frame.New(frame.RECEIPT, frame.ReceiptId, f7.Header.Get(frame.Receipt)))
		require.NoError(t, err)
	}()

	var sub *Subscription
	var err error
	sub, err = conn.Subscribe("/queue/test-1", ackMode, opts...)

	require.NoError(t, err)
	require.NotNil(t, sub)

	for i := 1; i <= 5; i++ {
		msg := <-sub.C
		messageId := fmt.Sprintf("message-%d", i)
		bodyText := fmt.Sprintf("Message body %d", i)
		require.Equal(t, sub, msg.Subscription)
		require.Equal(t, []byte(bodyText), msg.Body)
		require.Equal(t, "/queue/test-1", msg.Destination)
		require.Equal(t, messageId, msg.Header.Get(frame.MessageId))
		if version == V12 && ackMode.ShouldAck() {
			require.Equal(t, messageId, msg.Header.Get(frame.Ack))
		}

		require.Equal(t, ackMode.ShouldAck(), msg.ShouldAck())
		if msg.ShouldAck() {
			err = msg.Conn.Ack(msg)
			require.NoError(t, err)
		}
	}

	err = sub.Unsubscribe(SubscribeOpt.Header("custom", "true"))
	require.NoError(t, err)

	err = conn.Disconnect()
	require.NoError(t, err)
}

func TestStompTransaction(t *testing.T) {

	ackModes := []AckMode{AckAuto, AckClient, AckClientIndividual}
	versions := []Version{V10, V11, V12}
	aborts := []bool{false, true}
	nacks := []bool{false, true}

	for _, ackMode := range ackModes {
		for _, version := range versions {
			for _, abort := range aborts {
				for _, nack := range nacks {
					subscribeTransactionHelper(t, ackMode, version, abort, nack)
				}
			}
		}
	}
}

func subscribeTransactionHelper(t *testing.T, ackMode AckMode, version Version, abort bool, nack bool) {
	conn, rw := connectHelper(t, version)
	stop := make(chan struct{})

	go func() {
		defer func() {
			rw.Close()
			close(stop)
		}()

		f3, err := rw.Read()
		require.NoError(t, err)
		require.Equal(t, "SUBSCRIBE", f3.Command)
		id, ok := f3.Header.Contains("id")
		require.True(t, ok)
		destination := f3.Header.Get("destination")
		require.Equal(t, "/queue/test-1", destination)
		ack := f3.Header.Get("ack")
		require.Equal(t, ackMode.String(), ack)

		for i := 1; i <= 5; i++ {
			messageId := fmt.Sprintf("message-%d", i)
			bodyText := fmt.Sprintf("Message body %d", i)
			f4 := frame.New("MESSAGE",
				frame.Subscription, id,
				frame.MessageId, messageId,
				frame.Destination, destination)
			if version == V12 && ackMode.ShouldAck() {
				f4.Header.Add(frame.Ack, messageId)
			}
			f4.Body = []byte(bodyText)
			err = rw.Write(f4)
			require.NoError(t, err)

			beginFrame, err := rw.Read()
			require.NoError(t, err)
			require.NotNil(t, beginFrame)
			require.Equal(t, "BEGIN", beginFrame.Command)
			tx, ok := beginFrame.Header.Contains(frame.Transaction)

			require.True(t, ok)

			if ackMode.ShouldAck() {
				f5, _ := rw.Read()
				if nack && version.SupportsNack() {
					require.Equal(t, "NACK", f5.Command)
				} else {
					require.Equal(t, "ACK", f5.Command)
				}
				if version == V12 {
					require.Equal(t, messageId, f5.Header.Get(frame.Id))
				} else {
					require.Equal(t, id, f5.Header.Get("subscription"))
					require.Equal(t, messageId, f5.Header.Get("message-id"))
				}
				require.Equal(t, tx, f5.Header.Get("transaction"))
			}

			sendFrame, _ := rw.Read()
			require.NotNil(t, sendFrame)
			require.Equal(t, "SEND", sendFrame.Command)
			require.Equal(t, tx, sendFrame.Header.Get("transaction"))

			commitFrame, _ := rw.Read()
			require.NotNil(t, commitFrame)
			if abort {
				require.Equal(t, "ABORT", commitFrame.Command)
			} else {
				require.Equal(t, "COMMIT", commitFrame.Command)
			}
			require.Equal(t, tx, commitFrame.Header.Get("transaction"))
		}

		f6, _ := rw.Read()
		require.Equal(t, "UNSUBSCRIBE", f6.Command)
		require.NotEqual(t, "", f6.Header.Get(frame.Receipt))
		require.Equal(t, id, f6.Header.Get(frame.Id))
		err = rw.Write(frame.New(frame.RECEIPT, frame.ReceiptId, f6.Header.Get(frame.Receipt)))
		require.NoError(t, err)

		f7, _ := rw.Read()
		require.Equal(t, "DISCONNECT", f7.Command)
		err = rw.Write(frame.New(frame.RECEIPT, frame.ReceiptId, f7.Header.Get(frame.Receipt)))
		require.NoError(t, err)
	}()

	sub, err := conn.Subscribe("/queue/test-1", ackMode)
	require.NoError(t, err)
	require.NotNil(t, sub)

	for i := 1; i <= 5; i++ {
		msg := <-sub.C
		messageId := fmt.Sprintf("message-%d", i)
		bodyText := fmt.Sprintf("Message body %d", i)
		require.Equal(t, sub, msg.Subscription)
		require.Equal(t, []byte(bodyText), msg.Body)
		require.Equal(t, "/queue/test-1", msg.Destination)
		require.Equal(t, messageId, msg.Header.Get(frame.MessageId))

		require.Equal(t, ackMode.ShouldAck(), msg.ShouldAck())
		tx := msg.Conn.Begin()
		require.NotEqual(t, "", tx.Id())
		if msg.ShouldAck() {
			if nack && version.SupportsNack() {
				err = tx.Nack(msg)
				require.NoError(t, err)
			} else {
				err = tx.Ack(msg)
				require.NoError(t, err)
			}
		}
		err = tx.Send("/queue/another-queue", "text/plain", []byte(bodyText))
		require.NoError(t, err)
		if abort {
			err = tx.Abort()
			require.NoError(t, err)
		} else {
			err = tx.Commit()
			require.NoError(t, err)
		}
	}

	err = sub.Unsubscribe()
	require.NoError(t, err)

	err = conn.Disconnect()
	require.NoError(t, err)
}

func TestStompHeartBeatReadTimeout(t *testing.T) {
	conn, rw := createHeartBeatConnection(t, 100, 10000, time.Millisecond)

	go func() {
		f1, err := rw.Read()
		require.NoError(t, err)
		require.Equal(t, "SUBSCRIBE", f1.Command)
		messageFrame := frame.New("MESSAGE",
			"destination", f1.Header.Get("destination"),
			"message-id", "1",
			"subscription", f1.Header.Get("id"))
		messageFrame.Body = []byte("Message body")
		err = rw.Write(messageFrame)
		require.NoError(t, err)
	}()

	sub, err := conn.Subscribe("/queue/test1", AckAuto)
	require.NoError(t, err)
	require.Equal(t, 101*time.Millisecond, conn.readTimeout)
	//println("read timeout", conn.readTimeout.String())

	msg, ok := <-sub.C
	require.NotNil(t, msg)
	require.True(t, ok)

	msg, ok = <-sub.C
	require.NotNil(t, msg)
	require.True(t, ok)
	require.Error(t, msg.Err)
	require.Equal(t, "read timeout", msg.Err.Error())

	msg, ok = <-sub.C
	require.Nil(t, msg)
	require.False(t, ok)

	stats := conn.Stats()
	require.Equal(t, int64(1), stats.WritesSent)
	require.Equal(t, int64(1), stats.ReadsReceived)
}

func TestStompHeartBeatWriteTimeout(t *testing.T) {
	t.Skip("not finished yet")
	conn, rw := createHeartBeatConnection(t, 10000, 100, time.Millisecond*1)

	go func() {
		f1, err := rw.Read()
		require.NoError(t, err)
		require.Nil(t, f1)

	}()

	time.Sleep(250)
	err := conn.Disconnect()
	require.NoError(t, err)
}

func createHeartBeatConnection(
	t testing.TB,
	readTimeout, writeTimeout int,
	readTimeoutError time.Duration) (*Conn, *fakeReaderWriter) {
	fc1, fc2 := testutil.NewFakeConn(t)
	stop := make(chan struct{})

	reader := frame.NewReader(fc2)
	writer := frame.NewWriter(fc2)

	go func() {
		f1, err := reader.Read()
		require.NoError(t, err)
		require.Equal(t, "CONNECT", f1.Command)
		require.Equal(t, "1,1", f1.Header.Get("heart-beat"))
		f2 := frame.New("CONNECTED", "version", "1.2")
		f2.Header.Add("heart-beat", fmt.Sprintf("%d,%d", readTimeout, writeTimeout))
		err = writer.Write(f2)
		require.NoError(t, err)
		close(stop)
	}()

	conn, err := Connect(fc1,
		ConnOpt.HeartBeat(time.Millisecond, time.Millisecond),
		ConnOpt.HeartBeatError(readTimeoutError),
		ConnOpt.WithStats())
	require.NoError(t, err)
	require.NotNil(t, conn)
	<-stop
	return conn, &fakeReaderWriter{
		reader: reader,
		writer: writer,
		conn:   fc2,
	}
}

// Testing Timeouts when receiving receipts
func sendFrameHelper(f *frame.Frame, c chan *frame.Frame) {
	c <- f
}

func TestStompTimeoutTriggers(t *testing.T) {
	const timeout = 1 * time.Millisecond
	f := frame.Frame{}
	request := writeRequest{
		Frame: &f,
		C:     make(chan *frame.Frame),
	}

	err := readReceiptWithTimeout(request.C, timeout, ErrMsgReceiptTimeout)

	require.Error(t, err)
}

func TestStompChannelReceviesReceipt(t *testing.T) {
	const timeout = 1 * time.Second
	f := frame.Frame{}
	request := writeRequest{
		Frame: &f,
		C:     make(chan *frame.Frame),
	}
	receipt := frame.Frame{
		Command: frame.RECEIPT,
	}

	go sendFrameHelper(&receipt, request.C)
	err := readReceiptWithTimeout(request.C, timeout, ErrMsgReceiptTimeout)

	require.NoError(t, err)
}

func TestStompChannelReceviesNonReceipt(t *testing.T) {
	const timeout = 1 * time.Second
	f := frame.Frame{}
	request := writeRequest{
		Frame: &f,
		C:     make(chan *frame.Frame),
	}
	receipt := frame.Frame{
		Command: "NOT A RECEIPT",
	}

	go sendFrameHelper(&receipt, request.C)
	err := readReceiptWithTimeout(request.C, timeout, ErrMsgReceiptTimeout)

	require.Error(t, err)
}

func TestStompZeroTimeout(t *testing.T) {
	const timeout = 0 * time.Second
	f := frame.Frame{}
	request := writeRequest{
		Frame: &f,
		C:     make(chan *frame.Frame),
	}
	receipt := frame.Frame{
		Command: frame.RECEIPT,
	}

	go sendFrameHelper(&receipt, request.C)
	err := readReceiptWithTimeout(request.C, timeout, ErrMsgReceiptTimeout)

	require.NoError(t, err)
}

func TestStompConnectWithContext(t *testing.T) {
	fc1, fc2 := testutil.NewFakeConn(t)

	go func() {
		buff := make([]byte, 1024)
		fc2.Read(buff)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	_, err := ConnectWithContext(ctx, fc1)
	// the err here is "io timeout" because the server did not reply to any stomp message
	// and the connection waited longer than the 5 seconds we set
	require.Error(t, err)
}
