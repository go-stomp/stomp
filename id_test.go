package stomp

import (
	//. "gopkg.in/check.v1"
	"runtime"
)

// only used during testing, does not need to be thread-safe
func resetId() {
	_lastId = 0
}

func (s *StompSuite) SetUpSuite(c *C) {
	resetId()
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func (s *StompSuite) TearDownSuite(c *C) {
	runtime.GOMAXPROCS(1)
}

func (s *StompSuite) TestAllocateId(c *C) {
	c.Assert(allocateId(), Equals, "1")
	c.Assert(allocateId(), Equals, "2")

	ch := make(chan bool, 50)
	for i := 0; i < 50; i++ {
		go doAllocate(100, ch)
	}

	for i := 0; i < 50; i++ {
		<-ch
	}

	c.Assert(allocateId(), Equals, "5003")
}

func doAllocate(count int, ch chan bool) {
	for i := 0; i < count; i++ {
		_ = allocateId()
	}
	ch <- true
}
