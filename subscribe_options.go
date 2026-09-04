package stomp

import (
	"github.com/go-stomp/stomp/v3/frame"
)

// SubscribeOpt contains options for for the Conn.Subscribe function.
var SubscribeOpt struct {
	// Id provides the opportunity to specify the value of the "id" header
	// entry in the STOMP SUBSCRIBE frame.
	//
	// If the client program does specify the value for "id",
	// it is responsible for choosing a unique value.
	Id func(id string) func(*frame.Frame) error

	// Header provides the opportunity to include custom header entries
	// in the SUBSCRIBE frame that the client sends to the server.
	Header func(key, value string) func(*frame.Frame) error

	// Receipt makes Conn.Subscribe wait for the server to confirm the
	// subscription with a RECEIPT frame before returning, avoiding a race
	// where a message published right after Subscribe() returns is lost
	// because the server hasn't finished creating the subscription yet.
	//
	// receiptId is the value of the "receipt" header sent to the server. If
	// left empty, a unique value is generated.
	//
	// If confirmation doesn't arrive within ConnOpt.SubscribeReceiptTimeout,
	// Subscribe unsubscribes again and returns ErrSubscribeReceiptTimeout.
	//
	// Reply-to (temporary queue) subscriptions are never sent to the server
	// and so cannot be confirmed: using Receipt with one makes Subscribe
	// return ErrReceiptNotSupportedForReplyTo.
	Receipt func(receiptId string) func(*frame.Frame) error
}

func init() {
	SubscribeOpt.Id = func(id string) func(*frame.Frame) error {
		return func(f *frame.Frame) error {
			if f.Command != frame.SUBSCRIBE {
				return ErrInvalidCommand
			}
			f.Header.Set(frame.Id, id)
			return nil
		}
	}

	SubscribeOpt.Header = func(key, value string) func(*frame.Frame) error {
		return func(f *frame.Frame) error {
			if f.Command != frame.SUBSCRIBE &&
				f.Command != frame.UNSUBSCRIBE {
				return ErrInvalidCommand
			}
			f.Header.Add(key, value)
			return nil
		}
	}

	SubscribeOpt.Receipt = func(receiptId string) func(*frame.Frame) error {
		return func(f *frame.Frame) error {
			if f.Command != frame.SUBSCRIBE {
				return ErrInvalidCommand
			}
			if receiptId == "" {
				receiptId = allocateId()
			}
			f.Header.Set(frame.Receipt, receiptId)
			return nil
		}
	}
}
