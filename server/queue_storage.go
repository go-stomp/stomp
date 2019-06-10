package server

import (
	"github.com/ingelectus/stomp/frame"
)

// QueueStorage is an interface that abstracts the queue storage mechanism.
// The intent is that different queue storage implementations can be
// used, depending on preference. Queue storage mechanisms could include
// in-memory, and various persistent storage mechanisms (eg file system, DB, etc).
type QueueStorage interface {
	// Enqueue adds a MESSAGE frame to the end of the queue.
	Enqueue(queue string, frame *frame.Frame) error

	// Requeue adds a MESSAGE frame to the head of the queue.
	// This will happen if a client fails to acknowledge receipt.
	Requeue(queue string, frame *frame.Frame) error

	// Dequeue removes a frame from the head of the queue.
	// Returns nil if no frame is available.
	Dequeue(queue string) (*frame.Frame, error)

	// Start is called at server startup. Allows the queue storage
	// to perform any initialization.
	Start()

	// Stop is called prior to server shutdown. Allows the queue storage
	// to perform any cleanup, such as flushing to disk.
	Stop()
}
