package topic

import (
	"testing"

	"github.com/go-stomp/stomp/v3/frame"
	"github.com/stretchr/testify/require"
)

func TestTopicWithoutSubscription(t *testing.T) {
	topic := newTopic("destination")

	f := frame.New(frame.MESSAGE,
		frame.Destination, "destination")

	topic.Enqueue(f)
}

func TestTopicWithOneSubscription(t *testing.T) {
	sub := &fakeSubscription{}

	topic := newTopic("destination")
	topic.Subscribe(sub)

	f := frame.New(frame.MESSAGE,
		frame.Destination, "destination")

	topic.Enqueue(f)

	require.Len(t, sub.Frames, 1)
	require.Equal(t, f, sub.Frames[0])
}

func TestTopicWithTwoSubscriptions(t *testing.T) {
	sub1 := &fakeSubscription{}
	sub2 := &fakeSubscription{}

	topic := newTopic("destination")
	topic.Subscribe(sub1)
	topic.Subscribe(sub2)

	f := frame.New(frame.MESSAGE,
		frame.Destination, "destination",
		"xxx", "yyy")

	topic.Enqueue(f)

	require.Len(t, sub1.Frames, 1)
	require.Len(t, sub2.Frames, 1)
	require.NotSame(t, f, sub1.Frames[0])
	require.Equal(t, f, sub2.Frames[0])
}

type fakeSubscription struct {
	// frames received by the subscription
	Frames []*frame.Frame
}

func (s *fakeSubscription) SendTopicFrame(f *frame.Frame) {
	s.Frames = append(s.Frames, f)
}
