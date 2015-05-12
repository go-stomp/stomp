package stomp

import (
	//. "gopkg.in/check.v1"
	"io"
	"strings"
	"testing/iotest"
)

type ReaderSuite struct{}

var _ = Suite(&ReaderSuite{})

func (s *ReaderSuite) TestConnect(c *C) {
	reader := NewReader(strings.NewReader("CONNECT\nlogin:xxx\npasscode:yyy\n\n\x00"))

	frame, err := reader.Read()
	c.Assert(err, IsNil)
	c.Assert(frame, NotNil)
	c.Assert(len(frame.Body), Equals, 0)

	// ensure we are at the end of input
	frame, err = reader.Read()
	c.Assert(frame, IsNil)
	c.Assert(err, Equals, io.EOF)
}

func (s *ReaderSuite) TestMultipleReads(c *C) {
	text := "SEND\ndestination:xxx\n\nPayload\x00\n" +
		"SEND\ndestination:yyy\ncontent-length:12\n\n" +
		"123456789AB\x00\x00"

	ioreaders := []io.Reader{
		strings.NewReader(text),
		iotest.DataErrReader(strings.NewReader(text)),
		iotest.OneByteReader(strings.NewReader(text)),
	}

	for _, ioreader := range ioreaders {
		// uncomment the following line to view the bytes being read
		//ioreader = iotest.NewReadLogger("RX", ioreader)
		reader := NewReader(ioreader)
		frame, err := reader.Read()
		c.Assert(err, IsNil)
		c.Assert(frame, NotNil)
		c.Assert(frame.Command, Equals, "SEND")
		c.Assert(frame.Header.Len(), Equals, 1)
		v := frame.Header.Get("destination")
		c.Assert(v, Equals, "xxx")
		c.Assert(string(frame.Body), Equals, "Payload")

		// now read a heart-beat from the input
		frame, err = reader.Read()
		c.Assert(err, IsNil)
		c.Assert(frame, IsNil)

		// this frame has content-length
		frame, err = reader.Read()
		c.Assert(err, IsNil)
		c.Assert(frame, NotNil)
		c.Assert(frame.Command, Equals, "SEND")
		c.Assert(frame.Header.Len(), Equals, 2)
		v = frame.Header.Get("destination")
		c.Assert(v, Equals, "yyy")
		n, ok, err := frame.ContentLength()
		c.Assert(n, Equals, 12)
		c.Assert(ok, Equals, true)
		c.Assert(err, IsNil)
		c.Assert(string(frame.Body), Equals, "123456789AB\x00")

		// ensure we are at the end of input
		frame, err = reader.Read()
		c.Assert(frame, IsNil)
		c.Assert(err, Equals, io.EOF)
	}
}

func (s *ReaderSuite) TestSendWithContentLength(c *C) {
	reader := NewReader(strings.NewReader("SEND\ndestination:xxx\ncontent-length:5\n\n\x00\x01\x02\x03\x04\x00"))

	frame, err := reader.Read()
	c.Assert(err, IsNil)
	c.Assert(frame, NotNil)
	c.Assert(frame.Command, Equals, "SEND")
	c.Assert(frame.Header.Len(), Equals, 2)
	v := frame.Header.Get("destination")
	c.Assert(v, Equals, "xxx")
	c.Assert(frame.Body, DeepEquals, []byte{0x00, 0x01, 0x02, 0x03, 0x04})

	// ensure we are at the end of input
	frame, err = reader.Read()
	c.Assert(frame, IsNil)
	c.Assert(err, Equals, io.EOF)
}

func (s *ReaderSuite) TestInvalidCommand(c *C) {
	reader := NewReader(strings.NewReader("sEND\ndestination:xxx\ncontent-length:5\n\n\x00\x01\x02\x03\x04\x00"))

	frame, err := reader.Read()
	c.Check(frame, IsNil)
	c.Assert(err, NotNil)
	c.Check(err.Error(), Equals, "invalid command")
}

func (s *ReaderSuite) TestMissingNull(c *C) {
	reader := NewReader(strings.NewReader("SEND\ndeestination:xxx\ncontent-length:5\n\n\x00\x01\x02\x03\x04\n"))

	f, err := reader.Read()
	c.Check(f, IsNil)
	c.Assert(err, NotNil)
	c.Check(err.Error(), Equals, "invalid frame format")
}

func (s *ReaderSuite) TestSubscribeWithoutId(c *C) {
	c.Skip("TODO: implement validate")

	reader := NewReader(strings.NewReader("SUBSCRIBE\ndestination:xxx\nIId:7\n\n\x00"))

	frame, err := reader.Read()
	c.Check(frame, IsNil)
	c.Assert(err, NotNil)
	c.Check(err.Error(), Equals, "missing header: id")
}

func (s *ReaderSuite) TestUnsubscribeWithoutId(c *C) {
	c.Skip("TODO: implement validate")

	reader := NewReader(strings.NewReader("UNSUBSCRIBE\nIId:7\n\n\x00"))

	frame, err := reader.Read()
	c.Check(frame, IsNil)
	c.Assert(err, NotNil)
	c.Check(err.Error(), Equals, "missing header: id")
}
