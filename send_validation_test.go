package stomp

import (
	"sync"
	"testing"

	"github.com/go-stomp/stomp/v3/frame"
	"github.com/stretchr/testify/require"
)

func TestSendBodyValidation(t *testing.T) {
	for _, transaction := range []bool{false, true} {
		c := &Conn{closeMutex: &sync.Mutex{}, writeCh: make(chan writeRequest, 1)}
		send := c.Send
		if transaction {
			send = (&Transaction{conn: c, id: "tx"}).Send
		}
		require.ErrorIs(t, send("/queue/a", "", []byte("a\x00b"), SendOpt.NoContentLength), frame.ErrInvalidFrameFormat)
		require.Empty(t, c.writeCh)
		require.NoError(t, send("/queue/a", "", []byte("a\x00b")))
		require.Equal(t, "3", (<-c.writeCh).Frame.Header.Get(frame.ContentLength))
		require.NoError(t, send("/queue/a", "", []byte("text"), SendOpt.NoContentLength))
		require.Equal(t, "", (<-c.writeCh).Frame.Header.Get(frame.ContentLength))
	}
}
