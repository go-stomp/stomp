package queue

import (
	"github.com/ingelectus/stomp/frame"
	. "gopkg.in/check.v1"
)

type MemoryQueueSuite struct{}

var _ = Suite(&MemoryQueueSuite{})

func (s *MemoryQueueSuite) Test1(c *C) {
	mq := NewMemoryQueueStorage()
	mq.Start()

	f1 := frame.New(frame.MESSAGE,
		frame.Destination, "/queue/test",
		frame.MessageId, "msg-001",
		frame.Subscription, "1")

	err := mq.Enqueue("/queue/test", f1)
	c.Assert(err, IsNil)

	f2 := frame.New(frame.MESSAGE,
		frame.Destination, "/queue/test",
		frame.MessageId, "msg-002",
		frame.Subscription, "1")

	err = mq.Enqueue("/queue/test", f2)
	c.Assert(err, IsNil)

	f3 := frame.New(frame.MESSAGE,
		frame.Destination, "/queue/test2",
		frame.MessageId, "msg-003",
		frame.Subscription, "2")

	err = mq.Enqueue("/queue/test2", f3)
	c.Assert(err, IsNil)

	// attempt to dequeue from a different queue
	f, err := mq.Dequeue("/queue/other-queue")
	c.Check(err, IsNil)
	c.Assert(f, IsNil)

	f, err = mq.Dequeue("/queue/test2")
	c.Check(err, IsNil)
	c.Assert(f, Equals, f3)

	f, err = mq.Dequeue("/queue/test")
	c.Check(err, IsNil)
	c.Assert(f, Equals, f1)

	f, err = mq.Dequeue("/queue/test")
	c.Check(err, IsNil)
	c.Assert(f, Equals, f2)

	f, err = mq.Dequeue("/queue/test")
	c.Check(err, IsNil)
	c.Assert(f, IsNil)

	f, err = mq.Dequeue("/queue/test2")
	c.Check(err, IsNil)
	c.Assert(f, IsNil)
}
