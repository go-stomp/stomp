package frame

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriterWrites(t *testing.T) {
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
		writeToBufferAndCheck(t, frameText)
	}
}

func TestWriterConnectHeadersAreNotEscaped(t *testing.T) {
	tests := []struct {
		name     string
		frame    *Frame
		expected string
	}{
		{
			name: "CONNECT",
			frame: New(CONNECT,
				"login", `user\name`,
				"passcode", `=x/xx-test:password:xx+xxx`,
			),
			expected: "CONNECT\n" +
				"login:user\\name\n" +
				"passcode:=x/xx-test:password:xx+xxx\n\n\x00",
		},
		{
			name:     "CONNECTED",
			frame:    New(CONNECTED, "server", `broker:1\main`),
			expected: "CONNECTED\nserver:broker:1\\main\n\n\x00",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var buffer bytes.Buffer
			err := NewWriter(&buffer).Write(test.frame)
			require.NoError(t, err)
			require.Equal(t, test.expected, buffer.String())
		})
	}
}

func TestWriterOtherFrameHeadersAreEscaped(t *testing.T) {
	for _, command := range []string{STOMP, SEND} {
		t.Run(command, func(t *testing.T) {
			var buffer bytes.Buffer
			err := NewWriter(&buffer).Write(New(command, "key", `abc:def\ghi`))
			require.NoError(t, err)
			require.Equal(t, command+"\nkey:abc\\cdef\\\\ghi\n\n\x00", buffer.String())
		})
	}
}

func TestWriterInvalidUnescapedHeadersAreRejected(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "empty key", key: "", value: "value"},
		{name: "colon in key", key: "bad:key", value: "value"},
		{name: "carriage return in key", key: "bad\rkey", value: "value"},
		{name: "line feed in key", key: "bad\nkey", value: "value"},
		{name: "NUL in key", key: "bad\x00key", value: "value"},
		{name: "carriage return in value", key: "key", value: "bad\rvalue"},
		{name: "line feed in value", key: "key", value: "bad\nvalue"},
		{name: "NUL in value", key: "key", value: "bad\x00value"},
	}

	for _, command := range []string{CONNECT, CONNECTED} {
		for _, test := range tests {
			t.Run(command+"/"+test.name, func(t *testing.T) {
				var buffer bytes.Buffer
				err := NewWriter(&buffer).Write(New(command, test.key, test.value))
				require.ErrorIs(t, err, ErrInvalidFrameFormat)
				require.Empty(t, buffer.String())
			})
		}
	}
}

func writeToBufferAndCheck(t *testing.T, frameText string) {
	reader := NewReader(strings.NewReader(frameText))

	frame, err := reader.Read()
	require.NoError(t, err)
	require.NotNil(t, frame)

	var b bytes.Buffer
	var writer = NewWriter(&b)
	err = writer.Write(frame)
	require.NoError(t, err)
	newFrameText := b.String()
	require.Equal(t, frameText, newFrameText)
	require.Equal(t, frameText, b.String())
}
