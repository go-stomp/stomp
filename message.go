package stomp

import (
	"errors"
	"github.com/guotie/stomp/frame"
	"strconv"
)

// A Message represents a message received from the STOMP server.
// In most cases a message corresponds to a single STOMP MESSAGE frame
// received from the STOMP server. If, however, the Err field is non-nil,
// then the message corresponds to a STOMP ERROR frame, or a connection
// error between the client and the server.
type Message struct {
	// Indicates whether an error was received on the subscription.
	// The error will contain details of the error. If the server
	// sent an ERROR frame, then the Body, ContentType and Header fields
	// will be populated according to the contents of the ERROR frame.
	Err error

	// Destination the message is sent to. The STOMP server should
	// in turn send this message to a STOMP client that has subscribed
	// to the destination.
	Destination string

	// MIME content type.
	ContentType string // MIME content

	// Connection that the message was received on.
	Conn *Conn

	// Subscription associated with the message.
	Subscription *Subscription

	// Optional header entries. When received from the server,
	// these are the header entries received with the message.
	// When sending to the server, these are optional header entries
	// that accompany the message to its destination.
	*Header

	// The message body, which is an arbitrary sequence of bytes.
	// The ContentType indicates the format of this body.
	Body []byte // Content of message
}

// ShouldAck returns true if this message should be acknowledged to
// the STOMP server that sent it.
func (msg *Message) ShouldAck() bool {
	if msg.Subscription == nil {
		// not received from the server, so no acknowledgement required
		return false
	}

	return msg.Subscription.AckMode() != AckAuto
}

func (msg *Message) createSendFrame() (*Frame, error) {
	if msg.Destination == "" {
		return nil, errors.New("no destination specififed")
	}
	f := NewFrame(frame.SEND, frame.Destination, msg.Destination)
	if msg.ContentType != "" {
		f.Set(frame.ContentType, msg.ContentType)
	}
	f.Set(frame.ContentLength, strconv.Itoa(len(msg.Body)))
	f.Body = msg.Body

	if msg.Header != nil {
		for i := 0; i < msg.Header.Len(); i++ {
			key, value := msg.Header.GetAt(i)
			f.Add(key, value)
		}
	}

	return f, nil
}
