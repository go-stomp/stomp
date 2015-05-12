package stomp

import (
	"fmt"
	"gopkg.in/stomp.v1/frame"
	"log"
)

// The Subscription type represents a client subscription to
// a destination. The subscription is created by calling Conn.Subscribe.
//
// Once a client has subscribed, it can receive messages from the C channel.
type Subscription struct {
	C           chan *Message
	id          string
	destination string
	conn        *Conn
	ackMode     AckMode
	completed   bool
}

// BUG(jpj): If the client does not read messages from the Subscription.C
// channel quickly enough, the client will stop reading messages from the
// server.

// Identification for this subscription. Unique among
// all subscriptions for the same Client.
func (s *Subscription) Id() string {
	return s.id
}

// Destination for which the subscription applies.
func (s *Subscription) Destination() string {
	return s.destination
}

// AckMode returns the Acknowledgement mode specified when the
// subscription was created.
func (s *Subscription) AckMode() AckMode {
	return s.ackMode
}

// Active returns whether the subscription is still active.
// Returns false if the subscription has been unsubscribed.
func (s *Subscription) Active() bool {
	return !s.completed
}

// Unsubscribes and closes the channel C.
func (s *Subscription) Unsubscribe() error {
	if s.completed {
		return completedSubscription
	}
	f := NewFrame(frame.UNSUBSCRIBE, frame.Id, s.id)
	s.conn.sendFrame(f)
	s.completed = true
	close(s.C)
	return nil
}
// Unsubscribes and closes the channel C without block
func (s *Subscription) UnsubscribeNonblock() error {
	if s.completed {
		return completedSubscription
	}
	//f := NewFrame(frame.UNSUBSCRIBE, frame.Id, s.id)
	//s.conn.sendFrame(f)
	s.completed = true
	close(s.C)
	return nil
}
// Read a message from the subscription. This is a convenience
// method: many callers will prefer to read from the channel C
// directly.
func (s *Subscription) Read() (*Message, error) {
	if s.completed {
		return nil, completedSubscription
	}
	msg, ok := <-s.C
	if !ok {
		return nil, completedSubscription
	}
	if msg.Err != nil {
		return nil, msg.Err
	}
	return msg, nil
}

func (s *Subscription) readLoop(ch chan *Frame) {
	for {
		f, ok := <-ch
		if !ok {
			return
		}

		if f.Command == frame.MESSAGE {
			destination := f.Get(frame.Destination)
			contentType := f.Get(frame.ContentType)
			msg := &Message{
				Destination:  destination,
				ContentType:  contentType,
				Conn:         s.conn,
				Subscription: s,
				Header:       f.Header,
				Body:         f.Body,
			}
			s.C <- msg
		} else if f.Command == frame.ERROR {
			message, _ := f.Contains(frame.Message)
			text := fmt.Sprintf("Subscription %s: %s: ERROR message:%s",
				s.id,
				s.destination,
				message)
			log.Println(text)
			contentType := f.Get(frame.ContentType)
			msg := &Message{
				Err: &Error{
					Message: f.Get(frame.Message),
					Frame:   f,
				},
				ContentType:  contentType,
				Conn:         s.conn,
				Subscription: s,
				Header:       f.Header,
				Body:         f.Body,
			}
			s.C <- msg
			s.completed = true
			close(s.C)
			return
		}
	}
}
