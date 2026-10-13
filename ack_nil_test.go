package stomp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAckNackNilMessage(t *testing.T) {
	for _, version := range []Version{V10, V11, V12} {
		t.Run(string(version), func(t *testing.T) {
			conn := &Conn{version: version}
			require.NotPanics(t, func() {
				require.ErrorIs(t, conn.Ack(nil), ErrNotReceivedMessage)
			})
			require.NotPanics(t, func() {
				if version == V10 {
					require.ErrorIs(t, conn.Nack(nil), ErrNackNotSupported)
				} else {
					require.ErrorIs(t, conn.Nack(nil), ErrNotReceivedMessage)
				}
			})
		})
	}
}
