package server

import (
	"fmt"
	"net"
	"runtime"
	"testing"
	"time"

	"github.com/go-stomp/stomp/v3"
	. "gopkg.in/check.v1"
)

func TestServer(t *testing.T) {
	TestingT(t)
}

type ServerSuite struct{}

var _ = Suite(&ServerSuite{})

func (s *ServerSuite) SetUpSuite(c *C) {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func (s *ServerSuite) TearDownSuite(c *C) {
	runtime.GOMAXPROCS(1)
}

func (s *ServerSuite) TestConnectAndDisconnect(c *C) {
	addr := ":59091"
	l, err := net.Listen("tcp", addr)
	c.Assert(err, IsNil)
	defer func() { _ = l.Close() }()
	go func() { _ = Serve(l) }()

	conn, err := net.Dial("tcp", "127.0.0.1"+addr)
	c.Assert(err, IsNil)

	client, err := stomp.Connect(conn)
	c.Assert(err, IsNil)

	err = client.Disconnect()
	c.Assert(err, IsNil)

	_ = conn.Close()
}


func (s *ServerSuite) TestHeartBeatingTolerance(c *C) {
	// Heart beat should not close connection exactly after not receiving message after cx
	//  it should add a pretty decent amount of time to counter network delay of other timing issues
	l, err := net.Listen("tcp", `127.0.0.1:0`)
	c.Assert(err, IsNil)
	defer func() { _ = l.Close() }()
	serv := Server{
		Addr:          l.Addr().String(),
		Authenticator: nil,
		QueueStorage:  nil,
		HeartBeat:     50 * time.Millisecond,
	}
	go func() { _ = serv.Serve(l) }()

	conn, err := net.Dial("tcp", l.Addr().String())
	c.Assert(err, IsNil)
	defer func() { _ = conn.Close() }()

	client, err := stomp.Connect(conn,
		stomp.ConnOpt.HeartBeat(50*time.Millisecond, 50*time.Millisecond),
	)
	c.Assert(err, IsNil)
	defer func() { _ = client.Disconnect() }()

	time.Sleep(serv.HeartBeat * 20) // let it go for some time to allow client and server to exchange some heart beat

	// Ensure the server has not closed his readChannel
	err = client.Send("/topic/whatever", "text/plain", []byte("hello"))
	c.Assert(err, IsNil)
}

func (s *ServerSuite) TestSendToQueuesAndTopics(c *C) {
	ch := make(chan bool, 2)
	println("number cpus:", runtime.NumCPU())

	addr := ":59092"

	l, err := net.Listen("tcp", addr)
	c.Assert(err, IsNil)
	defer func() { _ = l.Close() }()
	go func() { _ = Serve(l) }()

	// channel to communicate that the go routine has started
	started := make(chan bool)

	count := 100
	go runReceiver(c, ch, count, "/topic/test-1", addr, started)
	<-started
	go runReceiver(c, ch, count, "/topic/test-1", addr, started)
	<-started
	go runReceiver(c, ch, count, "/topic/test-2", addr, started)
	<-started
	go runReceiver(c, ch, count, "/topic/test-2", addr, started)
	<-started
	go runReceiver(c, ch, count, "/topic/test-1", addr, started)
	<-started
	go runReceiver(c, ch, count, "/queue/test-1", addr, started)
	<-started
	go runSender(c, ch, count, "/queue/test-1", addr, started)
	<-started
	go runSender(c, ch, count, "/queue/test-2", addr, started)
	<-started
	go runReceiver(c, ch, count, "/queue/test-2", addr, started)
	<-started
	go runSender(c, ch, count, "/topic/test-1", addr, started)
	<-started
	go runReceiver(c, ch, count, "/queue/test-3", addr, started)
	<-started
	go runSender(c, ch, count, "/queue/test-3", addr, started)
	<-started
	go runSender(c, ch, count, "/queue/test-4", addr, started)
	<-started
	go runSender(c, ch, count, "/topic/test-2", addr, started)
	<-started
	go runReceiver(c, ch, count, "/queue/test-4", addr, started)
	<-started

	for i := 0; i < 15; i++ {
		<-ch
	}
}

func runSender(c *C, ch chan bool, count int, destination, addr string, started chan bool) {
	conn, err := net.Dial("tcp", "127.0.0.1"+addr)
	c.Assert(err, IsNil)

	client, err := stomp.Connect(conn)
	c.Assert(err, IsNil)

	started <- true

	for i := 0; i < count; i++ {
		err := client.Send(destination, "text/plain",
			[]byte(fmt.Sprintf("%s test message %d", destination, i)))
		c.Assert(err, IsNil)
		//println("sent", i)
	}

	ch <- true
}

func runReceiver(c *C, ch chan bool, count int, destination, addr string, started chan bool) {
	conn, err := net.Dial("tcp", "127.0.0.1"+addr)
	c.Assert(err, IsNil)

	client, err := stomp.Connect(conn)
	c.Assert(err, IsNil)

	sub, err := client.Subscribe(destination, stomp.AckAuto)
	c.Assert(err, IsNil)
	c.Assert(sub, NotNil)

	started <- true

	for i := 0; i < count; i++ {
		msg := <-sub.C
		expectedText := fmt.Sprintf("%s test message %d", destination, i)
		c.Assert(msg.Body, DeepEquals, []byte(expectedText))
		//println("received", i)
	}
	ch <- true
}
