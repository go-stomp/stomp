package client

import (
	"github.com/ingelectus/stomp/frame"
	. "gopkg.in/check.v1"
)

type TxStoreSuite struct{}

var _ = Suite(&TxStoreSuite{})

func (s *TxStoreSuite) TestDoubleBegin(c *C) {
	txs := txStore{}

	err := txs.Begin("tx1")
	c.Assert(err, IsNil)

	err = txs.Begin("tx1")
	c.Assert(err, Equals, txAlreadyInProgress)
}

func (s *TxStoreSuite) TestSuccessfulTx(c *C) {
	txs := txStore{}

	err := txs.Begin("tx1")
	c.Check(err, IsNil)

	err = txs.Begin("tx2")
	c.Assert(err, IsNil)

	f1 := frame.New(frame.MESSAGE,
		frame.Destination, "/queue/1")

	f2 := frame.New(frame.MESSAGE,
		frame.Destination, "/queue/2")

	f3 := frame.New(frame.MESSAGE,
		frame.Destination, "/queue/3")

	f4 := frame.New(frame.MESSAGE,
		frame.Destination, "/queue/4")

	err = txs.Add("tx1", f1)
	c.Assert(err, IsNil)
	err = txs.Add("tx1", f2)
	c.Assert(err, IsNil)
	err = txs.Add("tx1", f3)
	c.Assert(err, IsNil)
	err = txs.Add("tx2", f4)

	var tx1 []*frame.Frame

	txs.Commit("tx1", func(f *frame.Frame) error {
		tx1 = append(tx1, f)
		return nil
	})
	c.Check(err, IsNil)

	var tx2 []*frame.Frame

	err = txs.Commit("tx2", func(f *frame.Frame) error {
		tx2 = append(tx2, f)
		return nil
	})
	c.Check(err, IsNil)

	c.Check(len(tx1), Equals, 3)
	c.Check(tx1[0], Equals, f1)
	c.Check(tx1[1], Equals, f2)
	c.Check(tx1[2], Equals, f3)

	c.Check(len(tx2), Equals, 1)
	c.Check(tx2[0], Equals, f4)

	// already committed, so should cause an error
	err = txs.Commit("tx1", func(f *frame.Frame) error {
		c.Fatal("should not be called")
		return nil
	})
	c.Check(err, Equals, txUnknown)
}
