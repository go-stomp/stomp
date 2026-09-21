package stomp

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-stomp/stomp/v3/frame"
)

func TestSendOptReceiptWithoutConnReturnsError(t *testing.T) {
	f := frame.New(frame.SEND)
	err := SendOpt.Receipt(f)
	require.ErrorIs(t, err, ErrFrameHasNoConnection)
}

func TestSubscribeOptReceiptWithoutConnReturnsError(t *testing.T) {
	f := frame.New(frame.SUBSCRIBE)
	err := SubscribeOpt.Receipt("")(f)
	require.ErrorIs(t, err, ErrFrameHasNoConnection)
}
