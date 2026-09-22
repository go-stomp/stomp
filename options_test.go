package stomp

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-stomp/stomp/v3/frame"
)

func TestSendOptReceiptSetsPlaceholderWithoutConn(t *testing.T) {
	f := frame.New(frame.SEND)
	err := SendOpt.Receipt(f)
	require.NoError(t, err)
	id, _ := f.Header.Contains(frame.Receipt)
	require.Equal(t, pendingReceiptID, id)
}

func TestSubscribeOptReceiptSetsPlaceholderWithoutConn(t *testing.T) {
	f := frame.New(frame.SUBSCRIBE)
	err := SubscribeOpt.Receipt("")(f)
	require.NoError(t, err)
	id, _ := f.Header.Contains(frame.Receipt)
	require.Equal(t, pendingReceiptID, id)
}
