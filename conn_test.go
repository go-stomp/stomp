package stomp

import (
	"fmt"
	"io"
	"time"

	"github.com/guotie/stomp/frame"
	//. "gopkg.in/check.v1"
	"gopkg.in/stomp.v1/testutil"
)

type fakeReaderWriter struct {
	reader *Reader
	writer *Writer
	conn   io.ReadWriteCloser
}

func (rw *fakeReaderWriter) Read() (*Frame, error) {
	return rw.reader.Read()
}

func (rw *fakeReaderWriter) Write(f *Frame) error {
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

		reader := NewReader(fc2)
		writer := NewWriter(fc2)
		f1, err := reader.Read()
		c.Assert(err, IsNil)
		c.Assert(f1.Command, Equals, "CONNECT")
		f2 := NewFrame("ERROR", "message", "auth-failed")
		writer.Write(f2)
	}()

	conn, err := Connect(fc1, Options{})
	c.Assert(conn, IsNil)
	c.Assert(err, ErrorMatches, "auth-failed")
}

func (s *StompSuite) Test_successful_connect_and_disconnect(c *C) {
	testcases := []struct {
		Options           Options
		NegotiatedVersion string
		ExpectedVersion   Version
		ExpectedSession   string
		ExpectedHost      string
		ExpectedServer    string
	}{
		{
			Options:         Options{},
			ExpectedVersion: "1.0",
			ExpectedSession: "",
			ExpectedHost:    "the-server",
			ExpectedServer:  "some-server/1.1",
		},
		{
			Options:           Options{},
			NegotiatedVersion: "1.1",
			ExpectedVersion:   "1.1",
			ExpectedSession:   "the-session",
			ExpectedHost:      "the-server",
		},
		{
			Options:           Options{Host: "xxx"},
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
			reader := NewReader(fc2)
			writer := NewWriter(fc2)

			f1, err := reader.Read()
			c.Assert(err, IsNil)
			c.Assert(f1.Command, Equals, "CONNECT")
			host, _ := f1.Contains("host")
			c.Check(host, Equals, tc.ExpectedHost)
			connectedFrame := NewFrame("CONNECTED")
			if tc.NegotiatedVersion != "" {
				connectedFrame.Add("version", tc.NegotiatedVersion)
			}
			if tc.ExpectedSession != "" {
				connectedFrame.Add("session", tc.ExpectedSession)
			}
			if tc.ExpectedServer != "" {
				connectedFrame.Add("server", tc.ExpectedServer)
			}
			writer.Write(connectedFrame)

			f2, err := reader.Read()
			c.Assert(err, IsNil)
			c.Assert(f2.Command, Equals, "DISCONNECT")
			receipt, _ := f2.Contains("receipt")
			c.Check(receipt, Equals, "1")

			writer.Write(NewFrame("RECEIPT", frame.ReceiptId, "1"))
		}()

		client, err := Connect(fc1, tc.Options)
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
		reader := NewReader(fc2)
		writer := NewWriter(fc2)

		f1, err := reader.Read()
		c.Assert(err, IsNil)
		c.Assert(f1.Command, Equals, "CONNECT")
		c.Assert(f1.Get("login"), Equals, "guest")
		c.Assert(f1.Get("passcode"), Equals, "guest")
		c.Assert(f1.Get("host"), Equals, "/")
		c.Assert(f1.Get("x-max-length"), Equals, "50")
		connectedFrame := NewFrame("CONNECTED")
		connectedFrame.Add("session", "session-0voRHrG-VbBedx1Gwwb62Q")
		connectedFrame.Add("heart-beat", "0,0")
		connectedFrame.Add("server", "RabbitMQ/3.2.1")
		connectedFrame.Add("version", "1.0")
		writer.Write(connectedFrame)

		f2, err := reader.Read()
		c.Assert(err, IsNil)
		c.Assert(f2.Command, Equals, "DISCONNECT")
		receipt, _ := f2.Contains("receipt")
		c.Check(receipt, Equals, "1")

		writer.Write(NewFrame("RECEIPT", frame.ReceiptId, "1"))
	}()

	client, err := Connect(fc1, Options{
		Login:       "guest",
		Passcode:    "guest",
		Host:        "/",
		NonStandard: NewHeader("x-max-length", "50"),
	})
	c.Assert(err, IsNil)
	c.Assert(client, NotNil)
	c.Assert(client.Version(), Equals, V10)
	c.Assert(client.Session(), Equals, "session-0voRHrG-VbBedx1Gwwb62Q")
	c.Assert(client.Server(), Equals, "RabbitMQ/3.2.1")

	err = client.Disconnect()
	c.Assert(err, IsNil)

	<-stop
}

// Sets up a connection for testing
func connectHelper(c *C, version Version) (*Conn, *fakeReaderWriter) {
	fc1, fc2 := testutil.NewFakeConn(c)
	stop := make(chan struct{})

	reader := NewReader(fc2)
	writer := NewWriter(fc2)

	go func() {
		f1, err := reader.Read()
		c.Assert(err, IsNil)
		c.Assert(f1.Command, Equals, "CONNECT")
		f2 := NewFrame("CONNECTED", "version", version.String())
		writer.Write(f2)
		close(stop)
	}()

	conn, err := Connect(fc1, Options{})
	c.Assert(conn, NotNil)
	c.Assert(err, IsNil)
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
	header := NewHeader("id", "client-1", "custom", "true")

	for _, ackMode := range ackModes {
		for _, version := range versions {
			subscribeHelper(c, ackMode, version, nil)
			subscribeHelper(c, ackMode, version, header)
		}
	}
}

func subscribeHelper(c *C, ackMode AckMode, version Version, header *Header) {
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

		id, ok := f3.Contains("id")
		c.Assert(ok, Equals, true)

		if header != nil {
			for i := 0; i < header.Len(); i++ {
				k, v := header.GetAt(i)
				headerV, _ := f3.Contains(k)
				c.Assert(headerV, Equals, v)
			}
		}

		destination := f3.Get("destination")
		c.Assert(destination, Equals, "/queue/test-1")
		ack := f3.Get("ack")
		c.Assert(ack, Equals, ackMode.String())

		for i := 1; i <= 5; i++ {
			messageId := fmt.Sprintf("message-%d", i)
			bodyText := fmt.Sprintf("Message body %d", i)
			f4 := NewFrame("MESSAGE",
				frame.Subscription, id,
				frame.MessageId, messageId,
				frame.Destination, destination)
			if version == V12 {
				f4.Add(frame.Ack, messageId)
			}
			f4.Body = []byte(bodyText)
			rw.Write(f4)

			if ackMode.ShouldAck() {
				f5, _ := rw.Read()
				c.Assert(f5.Command, Equals, "ACK")
				if version == V12 {
					c.Assert(f5.Get("id"), Equals, messageId)
				} else {
					c.Assert(f5.Get("subscription"), Equals, id)
					c.Assert(f5.Get("message-id"), Equals, messageId)
				}
			}
		}

		f6, _ := rw.Read()
		c.Assert(f6.Command, Equals, "UNSUBSCRIBE")
		c.Assert(f6.Get(frame.Receipt), Not(Equals), "")
		c.Assert(f6.Get(frame.Id), Equals, id)
		rw.Write(NewFrame(frame.RECEIPT,
			frame.ReceiptId, f6.Get(frame.Receipt)))

		f7, _ := rw.Read()
		c.Assert(f7.Command, Equals, "DISCONNECT")
		rw.Write(NewFrame(frame.RECEIPT,
			frame.ReceiptId, f7.Get(frame.Receipt)))
	}()

	var sub *Subscription
	var err error
	if header != nil {
		sub, err = conn.SubscribeWithHeaders("/queue/test-1", ackMode, header)
	} else {
		sub, err = conn.Subscribe("/queue/test-1", ackMode)
	}

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

	err = sub.Unsubscribe()
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
	//println("subscribeTransactionHelper(", ackMode.String(), version, abort, nack, ")")
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
		id, ok := f3.Contains("id")
		c.Assert(ok, Equals, true)
		destination := f3.Get("destination")
		c.Assert(destination, Equals, "/queue/test-1")
		ack := f3.Get("ack")
		c.Assert(ack, Equals, ackMode.String())

		for i := 1; i <= 5; i++ {
			messageId := fmt.Sprintf("message-%d", i)
			bodyText := fmt.Sprintf("Message body %d", i)
			f4 := NewFrame("MESSAGE",
				frame.Subscription, id,
				frame.MessageId, messageId,
				frame.Destination, destination)
			if version == V12 {
				f4.Add(frame.Ack, messageId)
			}
			f4.Body = []byte(bodyText)
			rw.Write(f4)

			beginFrame, err := rw.Read()
			c.Assert(err, IsNil)
			c.Assert(beginFrame, NotNil)
			c.Check(beginFrame.Command, Equals, "BEGIN")
			tx, ok := beginFrame.Contains(frame.Transaction)

			c.Assert(ok, Equals, true)

			if ackMode.ShouldAck() {
				f5, _ := rw.Read()
				if nack && version.SupportsNack() {
					c.Assert(f5.Command, Equals, "NACK")
				} else {
					c.Assert(f5.Command, Equals, "ACK")
				}
				if version == V12 {
					c.Assert(f5.Get("id"), Equals, messageId)
				} else {
					c.Assert(f5.Get("subscription"), Equals, id)
					c.Assert(f5.Get("message-id"), Equals, messageId)
				}
				c.Assert(f5.Get("transaction"), Equals, tx)
			}

			sendFrame, _ := rw.Read()
			c.Assert(sendFrame, NotNil)
			c.Assert(sendFrame.Command, Equals, "SEND")
			c.Assert(sendFrame.Get("transaction"), Equals, tx)

			commitFrame, _ := rw.Read()
			c.Assert(commitFrame, NotNil)
			if abort {
				c.Assert(commitFrame.Command, Equals, "ABORT")
			} else {
				c.Assert(commitFrame.Command, Equals, "COMMIT")
			}
			c.Assert(commitFrame.Get("transaction"), Equals, tx)
		}

		f6, _ := rw.Read()
		c.Assert(f6.Command, Equals, "UNSUBSCRIBE")
		c.Assert(f6.Get(frame.Receipt), Not(Equals), "")
		c.Assert(f6.Get(frame.Id), Equals, id)
		rw.Write(NewFrame(frame.RECEIPT,
			frame.ReceiptId, f6.Get(frame.Receipt)))

		f7, _ := rw.Read()
		c.Assert(f7.Command, Equals, "DISCONNECT")
		rw.Write(NewFrame(frame.RECEIPT,
			frame.ReceiptId, f7.Get(frame.Receipt)))
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
		err = tx.Send("/queue/another-queue", "text/plain", []byte(bodyText), nil)
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
	conn, rw := createHeartBeatConnection(c, 100, 10000, time.Millisecond*1)

	go func() {
		f1, err := rw.Read()
		c.Assert(err, IsNil)
		c.Assert(f1.Command, Equals, "SUBSCRIBE")
		messageFrame := NewFrame("MESSAGE",
			"destination", f1.Get("destination"),
			"message-id", "1",
			"subscription", f1.Get("id"))
		messageFrame.Body = []byte("Message body")
		rw.Write(messageFrame)
	}()

	sub, err := conn.Subscribe("/queue/test1", AckAuto)
	c.Assert(err, IsNil)
	c.Check(conn.readTimeout, Equals, 101*time.Millisecond)
	println("read timeout", conn.readTimeout.String())

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
	c *C, readTimeout,
	writeTimeout int,
	readTimeoutError time.Duration) (*Conn, *fakeReaderWriter) {
	fc1, fc2 := testutil.NewFakeConn(c)
	stop := make(chan struct{})

	reader := NewReader(fc2)
	writer := NewWriter(fc2)

	go func() {
		f1, err := reader.Read()
		c.Assert(err, IsNil)
		c.Assert(f1.Command, Equals, "CONNECT")
		c.Assert(f1.Get("heart-beat"), Equals, "1,1")
		f2 := NewFrame("CONNECTED", "version", "1.2")
		f2.Add("heart-beat", fmt.Sprintf("%d,%d", readTimeout, writeTimeout))
		writer.Write(f2)
		close(stop)
	}()

	conn, err := Connect(fc1, Options{HeartBeat: "1,1", ReadHeartBeatError: readTimeoutError})
	c.Assert(conn, NotNil)
	c.Assert(err, IsNil)
	<-stop
	return conn, &fakeReaderWriter{
		reader: reader,
		writer: writer,
		conn:   fc2,
	}
}
