package stomp

import (
	"fmt"
	"io"
	"time"

	"github.com/ingelectus/stomp/frame"
	"github.com/ingelectus/stomp/testutil"
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
		writer.Write(f2)
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
			writer.Write(connectedFrame)

			f2, err := reader.Read()
			c.Assert(err, IsNil)
			c.Assert(f2.Command, Equals, "DISCONNECT")
			receipt, _ := f2.Header.Contains("receipt")
			c.Check(receipt, Equals, "1")

			writer.Write(frame.New("RECEIPT", frame.ReceiptId, "1"))
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
		writer.Write(connectedFrame)

		f2, err := reader.Read()
		c.Assert(err, IsNil)
		c.Assert(f2.Command, Equals, "DISCONNECT")
		receipt, _ := f2.Header.Contains("receipt")
		c.Check(receipt, Equals, "1")

		writer.Write(frame.New("RECEIPT", frame.ReceiptId, "1"))
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
		reader.Read()
		fc2.Write([]byte("\n"))
	}()

	client, err := Connect(fc1, ConnOpt.Host("the_server"))
	c.Assert(err, NotNil)
	c.Assert(client, IsNil)

	fc1.Close()
	<-stop
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
		writer.Write(f2)
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
			if version == V12 {
				f4.Header.Add(frame.Ack, messageId)
			}
			f4.Body = []byte(bodyText)
			rw.Write(f4)

			if ackMode.ShouldAck() {
				f5, _ := rw.Read()
				c.Assert(f5.Command, Equals, "ACK")
				if version == V12 {
					c.Assert(f5.Header.Get("id"), Equals, messageId)
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
		rw.Write(frame.New(frame.RECEIPT,
			frame.ReceiptId, f6.Header.Get(frame.Receipt)))

		f7, _ := rw.Read()
		c.Assert(f7.Command, Equals, "DISCONNECT")
		rw.Write(frame.New(frame.RECEIPT,
			frame.ReceiptId, f7.Header.Get(frame.Receipt)))
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

		c.Assert(msg.ShouldAck(), Equals, ackMode.ShouldAck())
		if msg.ShouldAck() {
			msg.Conn.Ack(msg)
		}
	}

	err = sub.Unsubscribe(SubscribeOpt.Header("custom", "true"))
	c.Assert(err, IsNil)

	conn.Disconnect()
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
			if version == V12 {
				f4.Header.Add(frame.Ack, messageId)
			}
			f4.Body = []byte(bodyText)
			rw.Write(f4)

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
					c.Assert(f5.Header.Get("id"), Equals, messageId)
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
		rw.Write(frame.New(frame.RECEIPT,
			frame.ReceiptId, f6.Header.Get(frame.Receipt)))

		f7, _ := rw.Read()
		c.Assert(f7.Command, Equals, "DISCONNECT")
		rw.Write(frame.New(frame.RECEIPT,
			frame.ReceiptId, f7.Header.Get(frame.Receipt)))
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
				tx.Nack(msg)
			} else {
				tx.Ack(msg)
			}
		}
		err = tx.Send("/queue/another-queue", "text/plain", []byte(bodyText))
		c.Assert(err, IsNil)
		if abort {
			tx.Abort()
		} else {
			tx.Commit()
		}
	}

	err = sub.Unsubscribe()
	c.Assert(err, IsNil)

	conn.Disconnect()
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
		rw.Write(messageFrame)
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
	c.Assert(msg.Err, NotNil)
	c.Assert(msg.Err.Error(), Equals, "read timeout")

	msg, ok = <-sub.C
	c.Assert(msg, IsNil)
	c.Assert(ok, Equals, false)
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
	conn.Disconnect()
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
		writer.Write(f2)
		close(stop)
	}()

	conn, err := Connect(fc1,
		ConnOpt.HeartBeat(time.Millisecond, time.Millisecond),
		ConnOpt.HeartBeatError(readTimeoutError))
	c.Assert(conn, NotNil)
	c.Assert(err, IsNil)
	<-stop
	return conn, &fakeReaderWriter{
		reader: reader,
		writer: writer,
		conn:   fc2,
	}
}
