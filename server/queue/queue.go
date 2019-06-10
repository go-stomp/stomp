/*
Package queue provides implementations of server-side queues.
*/
package queue

import (
	"github.com/ingelectus/stomp/frame"
	"github.com/ingelectus/stomp/server/client"
)

// Queue for storing message frames.
type Queue struct {
	destination string
	qstore      Storage
	subs        *client.SubscriptionList
}

// Create a new queue -- called from the queue manager only.
func newQueue(destination string, qstore Storage) *Queue {
	return &Queue{
		destination: destination,
		qstore:      qstore,
		subs:        client.NewSubscriptionList(),
	}
}

// Add a subscription to a queue. The subscription is removed
// whenever a frame is sent to the subscription and needs to
// be re-added when the subscription decides that the message
// has been received by the client.
func (q *Queue) Subscribe(sub *client.Subscription) error {
	// see if there is a frame available for this subscription
	f, err := q.qstore.Dequeue(sub.Destination())
	if err != nil {
		return err
	}
	if f == nil {
		// no frame available, so add to the subscription list
		q.subs.Add(sub)
	} else {
		// a frame is available, so send straight away without
		// adding the subscription to the list
		sub.SendQueueFrame(f)
	}
	return nil
}

// Unsubscribe a subscription.
func (q *Queue) Unsubscribe(sub *client.Subscription) {
	q.subs.Remove(sub)
}

// Send a message to the queue. If a subscription is available
// to receive the message, it is sent to the subscription without
// making it to the queue. Otherwise, the message is queued until
// a message is available.
func (q *Queue) Enqueue(f *frame.Frame) error {
	// find a subscription ready to receive the frame
	sub := q.subs.Get()
	if sub == nil {
		// no subscription available, add to the queue
		return q.qstore.Enqueue(q.destination, f)
	} else {
		// subscription is available, send it now without adding to queue
		sub.SendQueueFrame(f)
	}
	return nil
}

// Send a message to the front of the queue, probably because it
// failed to be sent to a client. If a subscription is available
// to receive the message, it is sent to the subscription without
// making it to the queue. Otherwise, the message is queued until
// a message is available.
func (q *Queue) Requeue(f *frame.Frame) error {
	// find a subscription ready to receive the frame
	sub := q.subs.Get()
	if sub == nil {
		// no subscription available, add to the queue
		return q.qstore.Requeue(q.destination, f)
	} else {
		// subscription is available, send it now without adding to queue
		sub.SendQueueFrame(f)
	}
	return nil
}
