package frame

import (
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/require"
)

func TestReaderConnect(t *testing.T) {
	reader := NewReader(strings.NewReader("CONNECT\nlogin:xxx\npasscode:yyy\n\n\x00"))

	frame, err := reader.Read()
	require.Nil(t, err)
	require.NotNil(t, frame)
	require.Len(t, frame.Body, 0)

	// ensure we are at the end of input
	frame, err = reader.Read()
	require.Nil(t, frame)
	require.ErrorIs(t, err, io.EOF)
}

func TestReaderMultipleReads(t *testing.T) {
	text := "SEND\ndestination:xxx\n\nPayload\x00\n" +
		"SEND\ndestination:yyy\ncontent-length:12\n" +
		"dodgy\\c\\n\\cheader:dodgy\\c\\n\\r\\nvalue\\  \\\n\n" +
		"123456789AB\x00\x00"

	ioreaders := []io.Reader{
		strings.NewReader(text),
		iotest.DataErrReader(strings.NewReader(text)),
		iotest.HalfReader(strings.NewReader(text)),
		iotest.OneByteReader(strings.NewReader(text)),
	}

	for _, ioreader := range ioreaders {
		// uncomment the following line to view the bytes being read
		//ioreader = iotest.NewReadLogger("RX", ioreader)
		reader := NewReader(ioreader)
		frame, err := reader.Read()
		require.NoError(t, err)
		require.NotNil(t, frame)
		require.Equal(t, "SEND", frame.Command)
		require.Equal(t, 1, frame.Header.Len())
		v := frame.Header.Get("destination")
		require.Equal(t, "xxx", v)
		require.Equal(t, "Payload", string(frame.Body))

		// now read a heart-beat from the input
		frame, err = reader.Read()
		require.NoError(t, err)
		require.Nil(t, frame)

		// this frame has content-length
		frame, err = reader.Read()
		require.NoError(t, err)
		require.NotNil(t, frame)
		require.Equal(t, "SEND", frame.Command)
		require.Equal(t, 3, frame.Header.Len())
		v = frame.Header.Get("destination")
		require.Equal(t, "yyy", v)
		n, ok, err := frame.Header.ContentLength()
		require.Equal(t, 12, n)
		require.True(t, ok)
		require.NoError(t, err)
		k, v := frame.Header.GetAt(2)
		require.Equal(t, "dodgy:\n:header", k)
		require.Equal(t, "dodgy:\n\r\nvalue\\  \\", v)
		require.Equal(t, "123456789AB\x00", string(frame.Body))

		// ensure we are at the end of input
		frame, err = reader.Read()
		require.Nil(t, frame)
		require.ErrorIs(t, err, io.EOF)
	}
}

func TestReaderSendWithContentLength(t *testing.T) {
	reader := NewReader(strings.NewReader("SEND\ndestination:xxx\ncontent-length:5\n\n\x00\x01\x02\x03\x04\x00"))

	frame, err := reader.Read()
	require.NoError(t, err)
	require.NotNil(t, frame)
	require.Equal(t, "SEND", frame.Command)
	require.Equal(t, 2, frame.Header.Len())
	v := frame.Header.Get("destination")
	require.Equal(t, "xxx", v)
	require.Equal(t, []byte{0x00, 0x01, 0x02, 0x03, 0x04}, frame.Body)

	// ensure we are at the end of input
	frame, err = reader.Read()
	require.Nil(t, frame)
	require.ErrorIs(t, err, io.EOF)
}

func TestReaderInvalidCommand(t *testing.T) {
	reader := NewReader(strings.NewReader("sEND\ndestination:xxx\ncontent-length:5\n\n\x00\x01\x02\x03\x04\x00"))

	frame, err := reader.Read()
	require.Nil(t, frame)
	require.Error(t, err)
	require.Equal(t, "invalid command", err.Error())
}

func TestReaderMissingNull(t *testing.T) {
	reader := NewReader(strings.NewReader("SEND\ndeestination:xxx\ncontent-length:5\n\n\x00\x01\x02\x03\x04\n"))

	f, err := reader.Read()
	require.Nil(t, f)
	require.Error(t, err)
	require.Equal(t, "invalid frame format", err.Error())
}

func TestReaderSubscribeWithoutId(t *testing.T) {
	t.Skip("TODO: implement validate")

	reader := NewReader(strings.NewReader("SUBSCRIBE\ndestination:xxx\nIId:7\n\n\x00"))

	frame, err := reader.Read()
	require.Nil(t, frame)
	require.Error(t, err)
	require.Equal(t, "missing header: id", err.Error())
}

func TestReaderUnsubscribeWithoutId(t *testing.T) {
	t.Skip("TODO: implement validate")

	reader := NewReader(strings.NewReader("UNSUBSCRIBE\nIId:7\n\n\x00"))

	frame, err := reader.Read()
	require.Nil(t, frame)
	require.Error(t, err)
	require.Equal(t, "missing header: id", err.Error())
}
