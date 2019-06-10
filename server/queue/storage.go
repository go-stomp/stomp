package queue

import (
	"github.com/ingelectus/stomp/frame"
)

// Interface for queue storage. The intent is that
// different queue storage implementations can be
// used, depending on preference. Queue storage
// mechanisms could include in-memory, and various
// persistent storage mechanisms (eg file system, DB, etc)
type Storage interface {
	// Pushes a MESSAGE frame to the end of the queue. Sets
	// the "message-id" header of the frame before adding to
	// the queue.
	Enqueue(queue string, frame *frame.Frame) error

	// Pushes a MESSAGE frame to the head of the queue. Sets
	// the "message-id" header of the frame if it is not
	// already set.
	Requeue(queue string, frame *frame.Frame) error

	// Removes a frame from the head of the queue.
	// Returns nil if no frame is available.
	Dequeue(queue string) (*frame.Frame, error)

	// Called at server startup. Allows the queue storage
	// to perform any initialization.
	Start()

	// Called prior to server shutdown. Allows the queue storage
	// to perform any cleanup.
	Stop()
}
