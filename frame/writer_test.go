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
