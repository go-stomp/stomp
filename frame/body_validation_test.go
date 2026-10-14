package frame

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriterValidatesBodyBeforeWriting(t *testing.T) {
	for _, f := range []*Frame{
		{Command: SEND, Body: []byte("a\x00b")},
		{Command: SEND, Header: NewHeader(ContentLength, "0"), Body: []byte("a")},
		{Command: SEND, Header: NewHeader(ContentLength, "2"), Body: []byte("a")},
		{Command: SEND, Header: NewHeader(ContentLength, "invalid")},
	} {
		var output bytes.Buffer
		writer := NewWriterSize(&output, 1)
		require.Error(t, writer.Write(f))
		require.Empty(t, output.Bytes())
		require.NoError(t, writer.Write(New(DISCONNECT)))
		require.Equal(t, "DISCONNECT\n\n\x00", output.String())
	}
}

func TestWriterBodyRoundTrip(t *testing.T) {
	for _, f := range []*Frame{
		{Command: SEND, Header: NewHeader(ContentLength, "3"), Body: []byte("a\x00b")},
		{Command: SEND, Body: []byte("abc")},
		{Command: SEND, Header: NewHeader(ContentLength, "0")},
	} {
		var output bytes.Buffer
		require.NoError(t, NewWriter(&output).Write(f))
		received, err := NewReader(&output).Read()
		require.NoError(t, err)
		require.Equal(t, string(f.Body), string(received.Body))
	}
}
