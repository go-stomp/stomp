/*
Package topic provides implementations of server-side topics.
*/
package topic

import (
	"container/list"

	"github.com/ingelectus/stomp/frame"
)

// A Topic is used for broadcasting to subscribed clients.
// In contrast to a queue, when a message is sent to a topic,
// that message is transmitted to all subscribed clients.
type Topic struct {
	destination string
	subs        *list.List
}

// Create a new topic -- called from the topic manager only.
func newTopic(destination string) *Topic {
	return &Topic{
		destination: destination,
		subs:        list.New(),
	}
}

// Subscribe adds a subscription to a topic. Any message sent to the
// topic will be transmitted to the subscription's client until
// unsubscription occurs.
func (t *Topic) Subscribe(sub Subscription) {
	t.subs.PushBack(sub)
}

// Unsubscribe causes a subscription to be removed from the topic.
func (t *Topic) Unsubscribe(sub Subscription) {
	for e := t.subs.Front(); e != nil; e = e.Next() {
		if sub == e.Value.(Subscription) {
			t.subs.Remove(e)
			return
		}
	}
}

// Enqueue send a message to the topic. All subscriptions receive a copy
// of the message.
func (t *Topic) Enqueue(f *frame.Frame) {
	switch t.subs.Len() {
	case 0:
	// no subscription, so do nothing

	case 1:
		// only one subscription, so can send the frame
		// without copying
		sub := t.subs.Front().Value.(Subscription)
		sub.SendTopicFrame(f)

	default:
		// more than one subscription, send clone for
		// all subscriptions except the last, which can
		// have the frame without copying
		for e := t.subs.Front(); e != nil; e = e.Next() {
			sub := e.Value.(Subscription)
			if e.Next() == nil {
				// the last in the list, send the frame
				// without copying
				sub.SendTopicFrame(f)
			} else {
				sub.SendTopicFrame(f.Clone())
			}
		}
	}
}
