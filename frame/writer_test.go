package frame

import (
	"bytes"
	"strings"

	. "gopkg.in/check.v1"
)

type WriterSuite struct{}

var _ = Suite(&WriterSuite{})

func (s *WriterSuite) TestWrites(c *C) {
	var frameTexts = []string{
		"CONNECT\nlogin:xxx\npasscode:yyy\n\n\x00",

		"SEND\n" +
			"destination:/queue/request\n" +
			"tx:1\n" +
			"content-length:5\n" +
			"\n\x00\x01\x02\x03\x04\x00",

		"SEND\ndestination:x\n\nABCD\x00",

		"SEND\ndestination:x\ndodgy\\nheader\\c:abc\\n\\c\n\n123456\x00",
	}

	for _, frameText := range frameTexts {
		writeToBufferAndCheck(c, frameText)
	}
}

func (s *WriterSuite) TestConnectHeadersAreNotEscaped(c *C) {
	tests := []struct {
		frame    *Frame
		expected string
	}{
		{
			frame: New(CONNECT,
				"login", `user\name`,
				"passcode", `=x/xx-test:password:xx+xxx`,
			),
			expected: "CONNECT\n" +
				"login:user\\name\n" +
				"passcode:=x/xx-test:password:xx+xxx\n\n\x00",
		},
		{
			frame:    New(CONNECTED, "server", `broker:1\main`),
			expected: "CONNECTED\nserver:broker:1\\main\n\n\x00",
		},
	}

	for _, test := range tests {
		var buffer bytes.Buffer
		err := NewWriter(&buffer).Write(test.frame)
		c.Assert(err, IsNil)
		c.Check(buffer.String(), Equals, test.expected)
	}
}

func (s *WriterSuite) TestOtherFrameHeadersAreEscaped(c *C) {
	for _, command := range []string{STOMP, SEND} {
		var buffer bytes.Buffer
		err := NewWriter(&buffer).Write(New(command, "key", `abc:def\ghi`))
		c.Assert(err, IsNil)
		c.Check(buffer.String(), Equals, command+"\nkey:abc\\cdef\\\\ghi\n\n\x00")
	}
}

func (s *WriterSuite) TestInvalidUnescapedHeadersAreRejected(c *C) {
	tests := []struct {
		key   string
		value string
	}{
		{key: "", value: "value"},
		{key: "bad:key", value: "value"},
		{key: "bad\rkey", value: "value"},
		{key: "bad\nkey", value: "value"},
		{key: "bad\x00key", value: "value"},
		{key: "key", value: "bad\rvalue"},
		{key: "key", value: "bad\nvalue"},
		{key: "key", value: "bad\x00value"},
	}

	for _, command := range []string{CONNECT, CONNECTED} {
		for _, test := range tests {
			var buffer bytes.Buffer
			err := NewWriter(&buffer).Write(New(command, test.key, test.value))
			c.Check(err, Equals, ErrInvalidFrameFormat)
			c.Check(buffer.String(), Equals, "")
		}
	}
}

func writeToBufferAndCheck(c *C, frameText string) {
	reader := NewReader(strings.NewReader(frameText))

	frame, err := reader.Read()
	c.Assert(err, IsNil)
	c.Assert(frame, NotNil)

	var b bytes.Buffer
	var writer = NewWriter(&b)
	err = writer.Write(frame)
	c.Assert(err, IsNil)
	newFrameText := b.String()
	c.Check(newFrameText, Equals, frameText)
	c.Check(b.String(), Equals, frameText)
}
