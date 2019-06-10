package client

import (
	"strconv"

	"github.com/ingelectus/stomp/frame"
)

// Opcode used in client requests.
type RequestOp int

func (r RequestOp) String() string {
	return strconv.Itoa(int(r))
}

// Valid value for client request opcodes.
const (
	SubscribeOp    RequestOp = iota // subscription ready
	UnsubscribeOp                   // subscription not ready
	EnqueueOp                       // send a message to a queue
	RequeueOp                       // re-queue a message, not successfully sent
	ConnectedOp                     // connection established
	DisconnectedOp                  // connection disconnected
)

// Client requests received to be processed by main processing loop
type Request struct {
	Op    RequestOp     // opcode for request
	Sub   *Subscription // SubscribeOp, UnsubscribeOp
	Frame *frame.Frame  // EnqueueOp, RequeueOp
	Conn  *Conn         // ConnectedOp, DisconnectedOp
}
