package stomp

import (
	"fmt"
	"io"
	"time"

	"github.com/go-stomp/stomp/v3/frame"
	"github.com/go-stomp/stomp/v3/testutil"

	"github.com/golang/mock/gomock"
	. "gopkg.in/check.v1"
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

func (s *StompSuite) Test_conn_option_set_logger(c *C) {
	fc1, fc2 := testutil.NewFakeConn(c)
	go func() {

		defer func() {
			fc2.Close()
			fc1.Close()
		}()

		reader := frame.NewReader(fc2)
		writer := frame.NewWriter(fc2)
		f1, err := reader.Read()
		c.Assert(err, IsNil)
		c.Assert(f1.Command, Equals, "CONNECT")
		f2 := frame.New("CONNECTED")
		err = writer.Write(f2)
		c.Assert(err, IsNil)
	}()

	ctrl := gomock.NewController(s.t)
	mockLogger := testutil.NewMockLogger(ctrl)

	conn, err := Connect(fc1, ConnOpt.Logger(mockLogger))
	c.Assert(err, IsNil)
	c.Check(conn, NotNil)

	c.Assert(conn.log, Equals, mockLogger)
}

func (s *StompSuite) Test_unsuccessful_connect(c *C) {
	fc1, fc2 := testutil.NewFakeConn(c)
	stop := make(chan struct{})

	go func() {
		defer func() {
			fc2.Close()
			close(stop)
		}()

		reader := frame.NewReader(fc2)
		writer := frame.NewWriter(fc2)
		f1, err := reader.Read()
		c.Assert(err, IsNil)
		c.Assert(f1.Command, Equals, "CONNECT")
		f2 := frame.New("ERROR", "message", "auth-failed")
		err = writer.Write(f2)
		c.Assert(err, IsNil)
	}()

	conn, err := Connect(fc1)
	c.Assert(conn, IsNil)
	c.Assert(err, ErrorMatches, "auth-failed")
}

func (s *StompSuite) Test_successful_connect_and_disconnect(c *C) {
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
		resetId()
		fc1, fc2 := testutil.NewFakeConn(c)
		stop := make(chan struct{})

		go func() {
			defer func() {
				fc2.Close()
				close(stop)
			}()
			reader := frame.NewReader(fc2)
			writer := frame.NewWriter(fc2)

			f1, err := reader.Read()
			c.Assert(err, IsNil)
			c.Assert(f1.Command, Equals, "CONNECT")
			host, _ := f1.Header.Contains("host")
			c.Check(host, Equals, tc.ExpectedHost)
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
			c.Assert(err, IsNil)

			f2, err := reader.Read()
			c.Assert(err, IsNil)
			c.Assert(f2.Command, Equals, "DISCONNECT")
			receipt, _ := f2.Header.Contains("receipt")
			c.Check(receipt, Equals, "1")

			err = writer.Write(frame.New("RECEIPT", frame.ReceiptId, "1"))
			c.Assert(err, IsNil)

		}()

		client, err := Connect(fc1, tc.Options...)
		c.Assert(err, IsNil)
		c.Assert(client, NotNil)
		c.Assert(client.Version(), Equals, tc.ExpectedVersion)
		c.Assert(client.Session(), Equals, tc.ExpectedSession)
		c.Assert(client.Server(), Equals, tc.ExpectedServer)

		err = client.Disconnect()
		c.Assert(err, IsNil)

		<-stop
	}
}

func (s *StompSuite) Test_successful_connect_get_headers(c *C) {
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
		resetId()
		fc1, fc2 := testutil.NewFakeConn(c)
		stop := make(chan struct{})

		go func() {
			defer func() {
				fc2.Close()
				close(stop)
			}()
			reader := frame.NewReader(fc2)
			writer := frame.NewWriter(fc2)

			f1, err := reader.Read()
			c.Assert(err, IsNil)
			c.Assert(f1.Command, Equals, "CONNECT")
			connectedFrame := frame.New("CONNECTED")
			for key, value := range tc.Headers {
				connectedFrame.Header.Add(key, value)
			}
			err = writer.Write(connectedFrame)
			c.Assert(err, IsNil)

			f2, err := reader.Read()
			c.Assert(err, IsNil)
			c.Assert(f2.Command, Equals, "DISCONNECT")
			receipt, _ := f2.Header.Contains("receipt")
			c.Check(receipt, Equals, "1")

			err = writer.Write(frame.New("RECEIPT", frame.ReceiptId, "1"))
			c.Assert(err, IsNil)

		}()

		client, err := Connect(fc1, tc.Options...)
		c.Assert(err, IsNil)
		c.Assert(client, NotNil)
		c.Assert(respHeaders, NotNil)
		for key, value := range tc.Headers {
			c.Assert(respHeaders.Get(key), Equals, value)
		}
		err = client.Disconnect()
		c.Assert(err, IsNil)

		<-stop
	}
}

func (s *StompSuite) Test_successful_connect_with_nonstandard_header(c *C) {
	resetId()
	fc1, fc2 := testutil.NewFakeConn(c)
	stop := make(chan struct{})

	go func() {
		defer func() {
			fc2.Close()
			close(stop)
		}()
		reader := frame.NewReader(fc2)
		writer := frame.NewWriter(fc2)

		f1, err := reader.Read()
		c.Assert(err, IsNil)
		c.Assert(f1.Command, Equals, "CONNECT")
		c.Assert(f1.Header.Get("login"), Equals, "guest")
		c.Assert(f1.Header.Get("passcode"), Equals, "guest")
		c.Assert(f1.Header.Get("host"), Equals, "/")
		c.Assert(f1.Header.Get("x-max-length"), Equals, "50")
		connectedFrame := frame.New("CONNECTED")
		connectedFrame.Header.Add("session", "session-0voRHrG-VbBedx1Gwwb62Q")
		connectedFrame.Header.Add("heart-beat", "0,0")
		connectedFrame.Header.Add("server", "RabbitMQ/3.2.1")
		connectedFrame.Header.Add("version", "1.0")
		err = writer.Write(connectedFrame)
		c.Assert(err, IsNil)

		f2, err := reader.Read()
		c.Assert(err, IsNil)
		c.Assert(f2.Command, Equals, "DISCONNECT")
		receipt, _ := f2.Header.Contains("receipt")
		c.Check(receipt, Equals, "1")

		err = writer.Write(frame.New("RECEIPT", frame.ReceiptId, "1"))
		c.Assert(err, IsNil)
	}()

	client, err := Connect(fc1,
		ConnOpt.Login("guest", "guest"),
		ConnOpt.Host("/"),
		ConnOpt.Header("x-max-length", "50"))
	c.Assert(err, IsNil)
	c.Assert(client, NotNil)
	c.Assert(client.Version(), Equals, V10)
	c.Assert(client.Session(), Equals, "session-0voRHrG-VbBedx1Gwwb62Q")
	c.Assert(client.Server(), Equals, "RabbitMQ/3.2.1")

	err = client.Disconnect()
	c.Assert(err, IsNil)

	<-stop
}

func (s *StompSuite) Test_connect_not_panic_on_empty_response(c *C) {
	resetId()
	fc1, fc2 := testutil.NewFakeConn(c)
	stop := make(chan struct{})

	go func() {
		defer func() {
			fc2.Close()
			close(stop)
		}()
		reader := frame.NewReader(fc2)
		_, err := reader.Read()
		c.Assert(err, IsNil)
		_, err = fc2.Write([]byte("\n"))
		c.Assert(err, IsNil)
	}()

	client, err := Connect(fc1, ConnOpt.Host("the_server"))
	c.Assert(err, NotNil)
	c.Assert(client, IsNil)

	fc1.Close()
	<-stop
}

func (s *StompSuite) Test_successful_disconnect_with_receipt_timeout(c *C) {
	resetId()
	fc1, fc2 := testutil.NewFakeConn(c)

	defer func() {
		fc2.Close()
	}()

	go func() {
		reader := frame.NewReader(fc2)
		writer := frame.NewWriter(fc2)

		f1, err := reader.Read()
		c.Assert(err, IsNil)
		c.Assert(f1.Command, Equals, "CONNECT")
		connectedFrame := frame.New("CONNECTED")
		err = writer.Write(connectedFrame)
		c.Assert(err, IsNil)
	}()

	client, err := Connect(fc1, ConnOpt.DisconnectReceiptTimeout(1 * time.Nanosecond))
	c.Assert(err, IsNil)
	c.Assert(client, NotNil)

	err = client.Disconnect()
	c.Assert(err, Equals, ErrDisconnectReceiptTimeout)
	c.Assert(client.closed, Equals, true)
}

// Sets up a connection for testing
func connectHelper(c *C, version Version) (*Conn, *fakeReaderWriter) {
	fc1, fc2 := testutil.NewFakeConn(c)
	stop := make(chan struct{})

	reader := frame.NewReader(fc2)
	writer := frame.NewWriter(fc2)

	go func() {
		f1, err := reader.Read()
		c.Assert(err, IsNil)
		c.Assert(f1.Command, Equals, "CONNECT")
		f2 := frame.New("CONNECTED", "version", version.String())
		err = writer.Write(f2)
		c.Assert(err, IsNil)
		close(stop)
	}()

	conn, err := Connect(fc1)
	c.Assert(err, IsNil)
	c.Assert(conn, NotNil)
	<-stop
	return conn, &fakeReaderWriter{
		reader: reader,
		writer: writer,
		conn:   fc2,
	}
}

func (s *StompSuite) Test_subscribe(c *C) {
	ackModes := []AckMode{AckAuto, AckClient, AckClientIndividual}
	versions := []Version{V10, V11, V12}

	for _, ackMode := range ackModes {
		for _, version := range versions {
			subscribeHelper(c, ackMode, version)
			subscribeHelper(c, ackMode, version,
				SubscribeOpt.Header("id", "client-1"),
				SubscribeOpt.Header("custom", "true"))
		}
	}
}

func subscribeHelper(c *C, ackMode AckMode, version Version, opts ...func(*frame.Frame) error) {
	conn, rw := connectHelper(c, version)
	stop := make(chan struct{})

	go func() {
		defer func() {
			rw.Close()
			close(stop)
		}()

		f3, err := rw.Read()
		c.Assert(err, IsNil)
		c.Assert(f3.Command, Equals, "SUBSCRIBE")

		id, ok := f3.Header.Contains("id")
		c.Assert(ok, Equals, true)

		destination := f3.Header.Get("destination")
		c.Assert(destination, Equals, "/queue/test-1")
		ack := f3.Header.Get("ack")
		c.Assert(ack, Equals, ackMode.String())

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
			c.Assert(err, IsNil)

			if ackMode.ShouldAck() {
				f5, _ := rw.Read()
				c.Assert(f5.Command, Equals, "ACK")
				if version == V12 {
					c.Assert(f5.Header.Get(frame.Id), Equals, messageId)
				} else {
					c.Assert(f5.Header.Get("subscription"), Equals, id)
					c.Assert(f5.Header.Get("message-id"), Equals, messageId)
				}
			}
		}

		f6, _ := rw.Read()
		c.Assert(f6.Command, Equals, "UNSUBSCRIBE")
		c.Assert(f6.Header.Get(frame.Receipt), Not(Equals), "")
		c.Assert(f6.Header.Get(frame.Id), Equals, id)
		err = rw.Write(frame.New(frame.RECEIPT, frame.ReceiptId, f6.Header.Get(frame.Receipt)))
		c.Assert(err, IsNil)

		f7, _ := rw.Read()
		c.Assert(f7.Command, Equals, "DISCONNECT")
		err = rw.Write(frame.New(frame.RECEIPT, frame.ReceiptId, f7.Header.Get(frame.Receipt)))
		c.Assert(err, IsNil)
	}()

	var sub *Subscription
	var err error
	sub, err = conn.Subscribe("/queue/test-1", ackMode, opts...)

	c.Assert(sub, NotNil)
	c.Assert(err, IsNil)

	for i := 1; i <= 5; i++ {
		msg := <-sub.C
		messageId := fmt.Sprintf("message-%d", i)
		bodyText := fmt.Sprintf("Message body %d", i)
		c.Assert(msg.Subscription, Equals, sub)
		c.Assert(msg.Body, DeepEquals, []byte(bodyText))
		c.Assert(msg.Destination, Equals, "/queue/test-1")
		c.Assert(msg.Header.Get(frame.MessageId), Equals, messageId)
		if version == V12 && ackMode.ShouldAck() {
			c.Assert(msg.Header.Get(frame.Ack), Equals, messageId)
		}

		c.Assert(msg.ShouldAck(), Equals, ackMode.ShouldAck())
		if msg.ShouldAck() {
			err = msg.Conn.Ack(msg)
			c.Assert(err, IsNil)
		}
	}

	err = sub.Unsubscribe(SubscribeOpt.Header("custom", "true"))
	c.Assert(err, IsNil)

	err = conn.Disconnect()
	c.Assert(err, IsNil)
}

func (s *StompSuite) TestTransaction(c *C) {

	ackModes := []AckMode{AckAuto, AckClient, AckClientIndividual}
	versions := []Version{V10, V11, V12}
	aborts := []bool{false, true}
	nacks := []bool{false, true}

	for _, ackMode := range ackModes {
		for _, version := range versions {
			for _, abort := range aborts {
				for _, nack := range nacks {
					subscribeTransactionHelper(c, ackMode, version, abort, nack)
				}
			}
		}
	}
}

func subscribeTransactionHelper(c *C, ackMode AckMode, version Version, abort bool, nack bool) {
	conn, rw := connectHelper(c, version)
	stop := make(chan struct{})

	go func() {
		defer func() {
			rw.Close()
			close(stop)
		}()

		f3, err := rw.Read()
		c.Assert(err, IsNil)
		c.Assert(f3.Command, Equals, "SUBSCRIBE")
		id, ok := f3.Header.Contains("id")
		c.Assert(ok, Equals, true)
		destination := f3.Header.Get("destination")
		c.Assert(destination, Equals, "/queue/test-1")
		ack := f3.Header.Get("ack")
		c.Assert(ack, Equals, ackMode.String())

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
			c.Assert(err, IsNil)

			beginFrame, err := rw.Read()
			c.Assert(err, IsNil)
			c.Assert(beginFrame, NotNil)
			c.Check(beginFrame.Command, Equals, "BEGIN")
			tx, ok := beginFrame.Header.Contains(frame.Transaction)

			c.Assert(ok, Equals, true)

			if ackMode.ShouldAck() {
				f5, _ := rw.Read()
				if nack && version.SupportsNack() {
					c.Assert(f5.Command, Equals, "NACK")
				} else {
					c.Assert(f5.Command, Equals, "ACK")
				}
				if version == V12 {
					c.Assert(f5.Header.Get(frame.Id), Equals, messageId)
				} else {
					c.Assert(f5.Header.Get("subscription"), Equals, id)
					c.Assert(f5.Header.Get("message-id"), Equals, messageId)
				}
				c.Assert(f5.Header.Get("transaction"), Equals, tx)
			}

			sendFrame, _ := rw.Read()
			c.Assert(sendFrame, NotNil)
			c.Assert(sendFrame.Command, Equals, "SEND")
			c.Assert(sendFrame.Header.Get("transaction"), Equals, tx)

			commitFrame, _ := rw.Read()
			c.Assert(commitFrame, NotNil)
			if abort {
				c.Assert(commitFrame.Command, Equals, "ABORT")
			} else {
				c.Assert(commitFrame.Command, Equals, "COMMIT")
			}
			c.Assert(commitFrame.Header.Get("transaction"), Equals, tx)
		}

		f6, _ := rw.Read()
		c.Assert(f6.Command, Equals, "UNSUBSCRIBE")
		c.Assert(f6.Header.Get(frame.Receipt), Not(Equals), "")
		c.Assert(f6.Header.Get(frame.Id), Equals, id)
		err = rw.Write(frame.New(frame.RECEIPT, frame.ReceiptId, f6.Header.Get(frame.Receipt)))
		c.Assert(err, IsNil)

		f7, _ := rw.Read()
		c.Assert(f7.Command, Equals, "DISCONNECT")
		err = rw.Write(frame.New(frame.RECEIPT, frame.ReceiptId, f7.Header.Get(frame.Receipt)))
		c.Assert(err, IsNil)
	}()

	sub, err := conn.Subscribe("/queue/test-1", ackMode)
	c.Assert(sub, NotNil)
	c.Assert(err, IsNil)

	for i := 1; i <= 5; i++ {
		msg := <-sub.C
		messageId := fmt.Sprintf("message-%d", i)
		bodyText := fmt.Sprintf("Message body %d", i)
		c.Assert(msg.Subscription, Equals, sub)
		c.Assert(msg.Body, DeepEquals, []byte(bodyText))
		c.Assert(msg.Destination, Equals, "/queue/test-1")
		c.Assert(msg.Header.Get(frame.MessageId), Equals, messageId)

		c.Assert(msg.ShouldAck(), Equals, ackMode.ShouldAck())
		tx := msg.Conn.Begin()
		c.Assert(tx.Id(), Not(Equals), "")
		if msg.ShouldAck() {
			if nack && version.SupportsNack() {
				err = tx.Nack(msg)
				c.Assert(err, IsNil)
			} else {
				err = tx.Ack(msg)
				c.Assert(err, IsNil)
			}
		}
		err = tx.Send("/queue/another-queue", "text/plain", []byte(bodyText))
		c.Assert(err, IsNil)
		if abort {
			err = tx.Abort()
			c.Assert(err, IsNil)
		} else {
			err = tx.Commit()
			c.Assert(err, IsNil)
		}
	}

	err = sub.Unsubscribe()
	c.Assert(err, IsNil)

	err = conn.Disconnect()
	c.Assert(err, IsNil)
}

func (s *StompSuite) TestHeartBeatReadTimeout(c *C) {
	conn, rw := createHeartBeatConnection(c, 100, 10000, time.Millisecond)

	go func() {
		f1, err := rw.Read()
		c.Assert(err, IsNil)
		c.Assert(f1.Command, Equals, "SUBSCRIBE")
		messageFrame := frame.New("MESSAGE",
			"destination", f1.Header.Get("destination"),
			"message-id", "1",
			"subscription", f1.Header.Get("id"))
		messageFrame.Body = []byte("Message body")
		err = rw.Write(messageFrame)
		c.Assert(err, IsNil)
	}()

	sub, err := conn.Subscribe("/queue/test1", AckAuto)
	c.Assert(err, IsNil)
	c.Check(conn.readTimeout, Equals, 101*time.Millisecond)
	//println("read timeout", conn.readTimeout.String())

	msg, ok := <-sub.C
	c.Assert(msg, NotNil)
	c.Assert(ok, Equals, true)

	msg, ok = <-sub.C
	c.Assert(msg, NotNil)
	c.Assert(ok, Equals, true)
	c.Assert(msg.Err, NotNil)
	c.Assert(msg.Err.Error(), Equals, "read timeout")

	msg, ok = <-sub.C
	c.Assert(msg, IsNil)
	c.Assert(ok, Equals, false)

	stats := conn.Stats()
	c.Assert(stats.WritesSent, Equals, int64(1))
	c.Assert(stats.ReadsReceived, Equals, int64(1))
}

func (s *StompSuite) TestHeartBeatWriteTimeout(c *C) {
	c.Skip("not finished yet")
	conn, rw := createHeartBeatConnection(c, 10000, 100, time.Millisecond*1)

	go func() {
		f1, err := rw.Read()
		c.Assert(err, IsNil)
		c.Assert(f1, IsNil)

	}()

	time.Sleep(250)
	err := conn.Disconnect()
	c.Assert(err, IsNil)
}

func createHeartBeatConnection(
	c *C,
	readTimeout, writeTimeout int,
	readTimeoutError time.Duration) (*Conn, *fakeReaderWriter) {
	fc1, fc2 := testutil.NewFakeConn(c)
	stop := make(chan struct{})

	reader := frame.NewReader(fc2)
	writer := frame.NewWriter(fc2)

	go func() {
		f1, err := reader.Read()
		c.Assert(err, IsNil)
		c.Assert(f1.Command, Equals, "CONNECT")
		c.Assert(f1.Header.Get("heart-beat"), Equals, "1,1")
		f2 := frame.New("CONNECTED", "version", "1.2")
		f2.Header.Add("heart-beat", fmt.Sprintf("%d,%d", readTimeout, writeTimeout))
		err = writer.Write(f2)
		c.Assert(err, IsNil)
		close(stop)
	}()

	conn, err := Connect(fc1,
		ConnOpt.HeartBeat(time.Millisecond, time.Millisecond),
		ConnOpt.HeartBeatError(readTimeoutError),
		ConnOpt.WithStats())
	c.Assert(conn, NotNil)
	c.Assert(err, IsNil)
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

//// GIVEN_TheTimeoutIsExceededBeforeTheReceiptIsReceived_WHEN_CallingReadReceiptWithTimeout_THEN_ReturnAnError
func (s *StompSuite) Test_TimeoutTriggers(c *C) {
	const timeout = 1 * time.Millisecond
	f := frame.Frame{}
	request := writeRequest{
		Frame: &f,
		C:     make(chan *frame.Frame),
	}

	err := readReceiptWithTimeout(request.C, timeout, ErrMsgReceiptTimeout)

	c.Assert(err, NotNil)
}

//// GIVEN_TheChannelReceivesTheReceiptBeforeTheTimeoutExpires_WHEN_CallingReadReceiptWithTimeout_THEN_DoNotReturnAnError
func (s *StompSuite) Test_ChannelReceviesReceipt(c *C) {
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

	c.Assert(err, IsNil)
}

//// GIVEN_TheChannelReceivesMessage_AND_TheMessageIsNotAReceipt_WHEN_CallingReadReceiptWithTimeout_THEN_ReturnAnError
func (s *StompSuite) Test_ChannelReceviesNonReceipt(c *C) {
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

	c.Assert(err, NotNil)
}

//// GIVEN_TheTimeoutIsSetToZero_AND_TheMessageIsReceived_WHEN_CallingReadReceiptWithTimeout_THEN_DoNotReturnAnError
func (s *StompSuite) Test_ZeroTimeout(c *C) {
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

	c.Assert(err, IsNil)
}
