package stomp

import (
//. "gopkg.in/check.v1"
)

type FrameSuite struct{}

var _ = Suite(&FrameSuite{})

func (s *FrameSuite) TestClone(c *C) {
	f1 := NewFrame("CONNECT", "login", "scott", "passcode", "leopard")
	f1.Body = []byte{1, 2, 3, 4}
	f2 := f1.Clone()
	f1.Set("login", "shaun")

	c.Check(f1.Get("login"), Equals, "shaun")
	c.Check(f2.Get("login"), Equals, "scott")
	c.Check(f2.Command, Equals, f1.Command)
	c.Check(f2.Get("passcode"), Equals, f1.Get("passcode"))
	c.Check(f2.Body, DeepEquals, f1.Body)

	// changing the body on one changes it for both
	f1.Body[0] = 99
	c.Check(f2.Body, DeepEquals, f1.Body)
}
