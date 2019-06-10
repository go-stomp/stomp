package queue

import (
	"container/list"

	"github.com/ingelectus/stomp/frame"
)

// In-memory implementation of the QueueStorage interface.
type MemoryQueueStorage struct {
	lists map[string]*list.List
}

func NewMemoryQueueStorage() Storage {
	m := &MemoryQueueStorage{lists: make(map[string]*list.List)}
	return m
}

func (m *MemoryQueueStorage) Enqueue(queue string, frame *frame.Frame) error {
	l, ok := m.lists[queue]
	if !ok {
		l = list.New()
		m.lists[queue] = l
	}
	l.PushBack(frame)

	return nil
}

// Pushes a frame to the head of the queue. Sets
// the "message-id" header of the frame if it is not
// already set.
func (m *MemoryQueueStorage) Requeue(queue string, frame *frame.Frame) error {
	l, ok := m.lists[queue]
	if !ok {
		l = list.New()
		m.lists[queue] = l
	}
	l.PushFront(frame)

	return nil
}

// Removes a frame from the head of the queue.
// Returns nil if no frame is available.
func (m *MemoryQueueStorage) Dequeue(queue string) (*frame.Frame, error) {
	l, ok := m.lists[queue]
	if !ok {
		return nil, nil
	}

	element := l.Front()
	if element == nil {
		return nil, nil
	}

	return l.Remove(element).(*frame.Frame), nil
}

// Called at server startup. Allows the queue storage
// to perform any initialization.
func (m *MemoryQueueStorage) Start() {
	m.lists = make(map[string]*list.List)
}

// Called prior to server shutdown. Allows the queue storage
// to perform any cleanup.
func (m *MemoryQueueStorage) Stop() {
	m.lists = nil
}
