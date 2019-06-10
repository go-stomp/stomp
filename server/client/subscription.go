package client

import (
	"github.com/ingelectus/stomp/frame"
)

type Subscription struct {
	conn    *Conn
	dest    string
	id      string            // client's subscription id
	ack     string            // auto, client, client-individual
	msgId   uint64            // message-id (or ack) for acknowledgement
	subList *SubscriptionList // am I in a list
	frame   *frame.Frame      // message allocated to subscription
}

func newSubscription(c *Conn, dest string, id string, ack string) *Subscription {
	return &Subscription{
		conn: c,
		dest: dest,
		id:   id,
		ack:  ack,
	}
}

func (s *Subscription) Destination() string {
	return s.dest
}

func (s *Subscription) Ack() string {
	return s.ack
}

func (s *Subscription) Id() string {
	return s.id
}

func (s *Subscription) IsAckedBy(msgId uint64) bool {
	switch s.ack {
	case frame.AckAuto:
		return true
	case frame.AckClient:
		// any later message acknowledges an earlier message
		return msgId >= s.msgId
	case frame.AckClientIndividual:
		return msgId == s.msgId
	}

	// should not get here
	panic("invalid value for subscript.ack")
}

func (s *Subscription) IsNackedBy(msgId uint64) bool {
	// TODO: not sure about this, interpreting NACK
	// to apply to an individual message
	return msgId == s.msgId
}

func (s *Subscription) SendQueueFrame(f *frame.Frame) {
	s.setSubscriptionHeader(f)
	s.frame = f

	// let the connection deal with the subscription
	// acknowledgement
	s.conn.subChannel <- s
}

// Send a message frame to the client, as part of this
// subscription. Called within the queue when a message
// frame is available.
func (s *Subscription) SendTopicFrame(f *frame.Frame) {
	s.setSubscriptionHeader(f)

	// topics are handled differently, they just go
	// straight to the client without acknowledgement
	s.conn.writeChannel <- f
}

func (s *Subscription) setSubscriptionHeader(f *frame.Frame) {
	if s.frame != nil {
		panic("subscription already has a frame pending")
	}
	f.Header.Set(frame.Subscription, s.id)
}
