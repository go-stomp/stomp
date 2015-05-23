package stomp

import (
	"github.com/guotie/stomp/frame"
)

var (
	invalidCommand        = newErrorMessage("invalid command")
	invalidFrameFormat    = newErrorMessage("invalid frame format")
	invalidVersion        = newErrorMessage("invalid version")
	completedTransaction  = newErrorMessage("transaction is completed")
	nackNotSupported      = newErrorMessage("NACK not supported in STOMP 1.0")
	notReceivedMessage    = newErrorMessage("cannot ack/nack a message, not from server")
	cannotNackAutoSub     = newErrorMessage("cannot send NACK for a subscription with ack:auto")
	completedSubscription = newErrorMessage("subscription is unsubscribed")
)

// StompError implements the Error interface, and provides
// additional information about a STOMP error.
type Error struct {
	Message string
	Frame   *Frame
}

func (e Error) Error() string {
	return e.Message
}

func missingHeader(name string) Error {
	return newErrorMessage("missing header: " + name)
}

func newErrorMessage(msg string) Error {
	return Error{Message: msg}
}

func newError(f *Frame) Error {
	e := Error{Frame: f}

	if f.Command == frame.ERROR {
		if message := f.Get(frame.Message); message != "" {
			e.Message = message
		} else {
			e.Message = "ERROR frame, missing message header"
		}
	} else {
		e.Message = "Unexpected frame: " + f.Command
	}
	return e
}
