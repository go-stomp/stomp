package client

import (
	"github.com/ingelectus/stomp"
	"github.com/ingelectus/stomp/frame"
	. "gopkg.in/check.v1"
)

type FrameSuite struct{}

var _ = Suite(&FrameSuite{})

func (s *FrameSuite) TestDetermineVersion_V10_Connect(c *C) {
	f := frame.New(frame.CONNECT)
	version, err := determineVersion(f)
	c.Check(err, IsNil)
	c.Check(version, Equals, stomp.V10)
}

func (s *FrameSuite) TestDetermineVersion_V10_Stomp(c *C) {
	// the "STOMP" command was introduced in V1.1, so it must
	// have an accept-version header
	f := frame.New(frame.STOMP)
	_, err := determineVersion(f)
	c.Check(err, Equals, missingHeader(frame.AcceptVersion))
}

func (s *FrameSuite) TestDetermineVersion_V11_Connect(c *C) {
	f := frame.New(frame.CONNECT)
	f.Header.Add(frame.AcceptVersion, "1.1")
	version, err := determineVersion(f)
	c.Check(version, Equals, stomp.V11)
	c.Check(err, IsNil)
}

func (s *FrameSuite) TestDetermineVersion_MultipleVersions(c *C) {
	f := frame.New(frame.CONNECT)
	f.Header.Add(frame.AcceptVersion, "1.2,1.1,1.0,2.0")
	version, err := determineVersion(f)
	c.Check(version, Equals, stomp.V12)
	c.Check(err, IsNil)
}

func (s *FrameSuite) TestDetermineVersion_IncompatibleVersions(c *C) {
	f := frame.New(frame.CONNECT)
	f.Header.Add(frame.AcceptVersion, "0.2,0.1,1.3,2.0")
	version, err := determineVersion(f)
	c.Check(version, Equals, stomp.Version(""))
	c.Check(err, Equals, unknownVersion)
}

func (s *FrameSuite) TestHeartBeat(c *C) {
	f := frame.New(frame.CONNECT,
		frame.AcceptVersion, "1.2",
		frame.Host, "XX")

	// no heart-beat header means zero values
	x, y, err := getHeartBeat(f)
	c.Check(x, Equals, 0)
	c.Check(y, Equals, 0)
	c.Check(err, IsNil)

	f.Header.Add("heart-beat", "123,456")
	x, y, err = getHeartBeat(f)
	c.Check(x, Equals, 123)
	c.Check(y, Equals, 456)
	c.Check(err, IsNil)

	f.Header.Set(frame.HeartBeat, "invalid")
	x, y, err = getHeartBeat(f)
	c.Check(x, Equals, 0)
	c.Check(y, Equals, 0)
	c.Check(err, Equals, invalidHeartBeat)

	f.Header.Del(frame.HeartBeat)
	_, _, err = getHeartBeat(f)
	c.Check(err, IsNil)

	f.Command = frame.SEND
	_, _, err = getHeartBeat(f)
	c.Check(err, Equals, invalidOperationForFrame)
}
