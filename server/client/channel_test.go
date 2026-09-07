package client

import (
	. "gopkg.in/check.v1"
)

// Test suite for testing that channels work the way I expect.
type ChannelSuite struct{}

var _ = Suite(&ChannelSuite{})

func (s *ChannelSuite) TestChannelWhenClosed(c *C) {

	ch := make(chan int, 10)

	ch <- 1
	ch <- 2

	select {
	case i, ok := <-ch:
		c.Assert(i, Equals, 1)
		c.Assert(ok, Equals, true)
	default:
		c.Error("expected value on channel")
	}

	select {
	case i := <-ch:
		c.Assert(i, Equals, 2)
	default:
		c.Error("expected value on channel")
	}

	select {
	case <-ch:
		c.Error("not expecting anything on the channel")
	default:
	}

	ch <- 3
	close(ch)

	select {
	case i := <-ch:
		c.Assert(i, Equals, 3)
	default:
		c.Error("expected value on channel")
	}

	select {
	case _, ok := <-ch:
		c.Assert(ok, Equals, false)
	default:
		c.Error("expected value on channel")
	}

	select {
	case _, ok := <-ch:
		c.Assert(ok, Equals, false)
	default:
		c.Error("expected value on channel")
	}
}

func (s *ChannelSuite) TestMultipleChannels(c *C) {

	ch1 := make(chan int, 10)
	ch2 := make(chan string, 10)

	ch1 <- 1

	select {
	case i, ok := <-ch1:
		c.Assert(i, Equals, 1)
		c.Assert(ok, Equals, true)
	case <-ch2:
	default:
		c.Error("expected value on channel")
	}

	select {
	case <-ch1:
		c.Error("not expected")
	case <-ch2:
		c.Error("not expected")
	default:
	}
}
