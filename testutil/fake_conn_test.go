package testutil

import (
	. "gopkg.in/check.v1"
	"testing"
)

func TestTestUtil(t *testing.T) {
	TestingT(t)
}

type FakeConnSuite struct{}

var _ = Suite(&FakeConnSuite{})

func (s *FakeConnSuite) TestFakeConn(c *C) {
	//c.Skip("temporary")
	fc1, fc2 := NewFakeConn(c)

	one := []byte{1, 2, 3, 4}
	two := []byte{5, 6, 7, 8, 9, 10, 11, 12, 13}
	stop := make(chan struct{})

	go func() {
		defer func() {
			_ = fc2.Close()
			close(stop)
		}()

		rx1 := make([]byte, 6)
		n, err := fc2.Read(rx1)
		c.Assert(n, Equals, 4)
		c.Assert(err, IsNil)
		c.Assert(rx1[0:n], DeepEquals, one)

		rx2 := make([]byte, 5)
		n, err = fc2.Read(rx2)
		c.Assert(n, Equals, 5)
		c.Assert(err, IsNil)
		c.Assert(rx2, DeepEquals, []byte{5, 6, 7, 8, 9})

		rx3 := make([]byte, 10)
		n, err = fc2.Read(rx3)
		c.Assert(n, Equals, 4)
		c.Assert(err, IsNil)
		c.Assert(rx3[0:n], DeepEquals, []byte{10, 11, 12, 13})
	}()

	c.Assert(fc1.C, Equals, c)
	c.Assert(fc2.C, Equals, c)

	n, err := fc1.Write(one)
	c.Assert(n, Equals, 4)
	c.Assert(err, IsNil)

	n, err = fc1.Write(two)
	c.Assert(n, Equals, len(two))
	c.Assert(err, IsNil)

	<-stop
}
