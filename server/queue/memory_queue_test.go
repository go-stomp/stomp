package queue

import (
	"testing"

	"github.com/go-stomp/stomp/v3/frame"
	"github.com/stretchr/testify/require"
)

func TestMemoryQueue(t *testing.T) {
	mq := NewMemoryQueueStorage()
	mq.Start()

	f1 := frame.New(frame.MESSAGE,
		frame.Destination, "/queue/test",
		frame.MessageId, "msg-001",
		frame.Subscription, "1")

	err := mq.Enqueue("/queue/test", f1)
	require.NoError(t, err)

	f2 := frame.New(frame.MESSAGE,
		frame.Destination, "/queue/test",
		frame.MessageId, "msg-002",
		frame.Subscription, "1")

	err = mq.Enqueue("/queue/test", f2)
	require.NoError(t, err)

	f3 := frame.New(frame.MESSAGE,
		frame.Destination, "/queue/test2",
		frame.MessageId, "msg-003",
		frame.Subscription, "2")

	err = mq.Enqueue("/queue/test2", f3)
	require.NoError(t, err)

	// attempt to dequeue from a different queue
	f, err := mq.Dequeue("/queue/other-queue")
	require.NoError(t, err)
	require.Nil(t, f)

	f, err = mq.Dequeue("/queue/test2")
	require.NoError(t, err)
	require.Equal(t, f3, f)

	f, err = mq.Dequeue("/queue/test")
	require.NoError(t, err)
	require.Equal(t, f1, f)

	f, err = mq.Dequeue("/queue/test")
	require.NoError(t, err)
	require.Equal(t, f2, f)

	f, err = mq.Dequeue("/queue/test")
	require.NoError(t, err)
	require.Nil(t, f)

	f, err = mq.Dequeue("/queue/test2")
	require.NoError(t, err)
	require.Nil(t, f)
}
