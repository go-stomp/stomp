package stomp

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-stomp/stomp/v3/frame"
)

const (
	subStateActive  = 0
	subStateClosing = 1
	subStateClosed  = 2
)

// The Subscription type represents a client subscription to
// a destination. The subscription is created by calling Conn.Subscribe.
//
// Once a client has subscribed, it can receive messages from the C channel.
type Subscription struct {
	C                         chan *Message
	id                        string
	replyToSet                bool
	destination               string
	conn                      *Conn
	ackMode                   AckMode
	state                     atomic.Int32
	done                      chan struct{}
	closeOnce                 sync.Once
	unsubscribeReceiptTimeout time.Duration
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
	return s.state.Load() == subStateActive
}

// Unsubscribes and closes the channel C.
func (s *Subscription) Unsubscribe(opts ...func(*frame.Frame) error) error {
	// transition to the "closing" state
	if !s.state.CompareAndSwap(subStateActive, subStateClosing) {
		return ErrCompletedSubscription
	}

	f := frame.New(frame.UNSUBSCRIBE, frame.Id, s.id)

	for _, opt := range opts {
		if opt == nil {
			return ErrNilOption
		}
		err := opt(f)
		if err != nil {
			return err
		}
	}

	if s.replyToSet {
		f.Header.Set(ReplyToHeader, s.id)
	}

	err := s.conn.sendFrame(f)
	if errors.Is(err, ErrClosedUnexpectedly) {
		msg := s.subscriptionErrorMessage("connection closed unexpectedly")
		s.closeChannel(msg)
		return err
	}

	// UNSUBSCRIBE is a bit weird in that it is tagged with a "receipt" header
	// on the I/O goroutine, so the above call to sendFrame() will not wait
	// for the resulting RECEIPT.
	//
	// We don't want to interfere with `s.C` since we might be "stealing"
	// MESSAGEs or ERRORs from another goroutine, so wait on `done` (closed
	// exactly once by closeChannel) for the terminal state transition instead.
	if s.unsubscribeReceiptTimeout <= 0 {
		<-s.done
		return nil
	}

	select {
	case <-s.done:
		return nil
	case <-time.After(s.unsubscribeReceiptTimeout):
		// s.done closing can race with the timeout firing; closeChannel closes
		// s.C before s.done, so re-check state to avoid sending on a closed s.C.
		if s.state.Load() == subStateClosed {
			return nil
		}
		msg := s.subscriptionErrorMessage("channel unsubscribe receipt timeout")
		s.C <- msg
		return &ErrUnsubscribeReceiptTimeout
	}
}

// abandon tears down a subscription that Conn.Subscribe created but never
// handed to the caller, because it failed while waiting for the confirming
// RECEIPT. Unlike Unsubscribe, it doesn't wait for the RECEIPT itself: there
// is nobody left to report the outcome to.
//
// If the broker never confirms the resulting UNSUBSCRIBE either, the drain
// goroutine started below (and C itself) is only reclaimed when the
// connection closes.
func (s *Subscription) abandon() {
	// transition to the "closing" state
	if !atomic.CompareAndSwapInt32(&s.state, subStateActive, subStateClosing) {
		return
	}

	// Nobody holds this subscription, so nobody will ever read C. Drain it
	// until readLoop closes it, or messages already in flight would fill C,
	// block readLoop on the frame channel and wedge processLoop with it.
	go func() {
		for range s.C {
		}
	}()

	f := frame.New(frame.UNSUBSCRIBE, frame.Id, s.id)
	if s.replyToSet {
		f.Header.Set(ReplyToHeader, s.id)
	}

	if err := s.conn.sendFrame(f); err != nil {
		// Nothing more we can do; the frame channel (and with it C) is closed
		// when the connection is torn down.
		s.conn.log.Infof("could not unsubscribe unconfirmed subscription %s: %s: %v",
			s.id, s.destination, err)
	}
}

// Read a message from the subscription. This is a convenience
// method: many callers will prefer to read from the channel C
// directly.
func (s *Subscription) Read() (*Message, error) {
	if !s.Active() {
		return nil, ErrCompletedSubscription
	}
	msg, ok := <-s.C
	if !ok {
		return nil, ErrCompletedSubscription
	}
	if msg.Err != nil {
		return nil, msg.Err
	}
	return msg, nil
}

func (s *Subscription) closeChannel(msg *Message) {
	s.closeOnce.Do(func() {
		if msg != nil {
			s.C <- msg
		}
		s.state.Store(subStateClosed)
		close(s.C)
		close(s.done)
	})
}

func (s *Subscription) subscriptionErrorMessage(message string) *Message {
	return &Message{
		Err: &Error{
			Message: fmt.Sprintf("Subscription %s: %s: %s", s.id, s.destination, message),
		},
	}
}

func (s *Subscription) readLoop(ch chan *frame.Frame) {
	for {
		f, ok := <-ch
		if !ok {
			state := s.state.Load()
			if state == subStateActive || state == subStateClosing {
				msg := s.subscriptionErrorMessage("channel read failed")
				s.closeChannel(msg)
			}
			return
		}

		switch f.Command {
		case frame.MESSAGE:
			destination := f.Header.Get(frame.Destination)
			contentType := f.Header.Get(frame.ContentType)
			msg := &Message{
				Destination:  destination,
				ContentType:  contentType,
				Conn:         s.conn,
				Subscription: s,
				Header:       f.Header,
				Body:         f.Body,
			}
			s.C <- msg
		case frame.ERROR:
			state := s.state.Load()
			if state == subStateActive || state == subStateClosing {
				message, _ := f.Header.Contains(frame.Message)
				text := fmt.Sprintf("Subscription %s: %s: ERROR message:%s",
					s.id,
					s.destination,
					message)
				s.conn.log.Info(text)
				contentType := f.Header.Get(frame.ContentType)
				msg := &Message{
					Err: &Error{
						Message: f.Header.Get(frame.Message),
						Frame:   f,
					},
					ContentType:  contentType,
					Conn:         s.conn,
					Subscription: s,
					Header:       f.Header,
					Body:         f.Body,
				}
				s.closeChannel(msg)
			}
			return
		case frame.RECEIPT:
			state := s.state.Load()
			if state == subStateActive || state == subStateClosing {
				s.closeChannel(nil)
			}
			return
		default:
			s.conn.log.Infof("Subscription %s: %s: unsupported frame type: %+v", s.id, s.destination, f)
		}
	}
}
