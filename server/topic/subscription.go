package topic

import (
	"github.com/ingelectus/stomp/frame"
)

// Subscription is the interface that wraps a subscriber to a topic.
type Subscription interface {
	// Send a message frame to the topic subscriber.
	SendTopicFrame(f *frame.Frame)
}
