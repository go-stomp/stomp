package server

import (
	"fmt"
	"net"
	"runtime"
	"testing"
	"time"

	"github.com/go-stomp/stomp/v3"
	"github.com/stretchr/testify/require"
)

func TestServerConnectAndDisconnect(t *testing.T) {
	addr := ":59091"
	l, err := net.Listen("tcp", addr)
	require.NoError(t, err)
	defer func() { l.Close() }()
	go Serve(l)

	conn, err := net.Dial("tcp", "127.0.0.1"+addr)
	require.NoError(t, err)

	client, err := stomp.Connect(conn)
	require.NoError(t, err)

	err = client.Disconnect()
	require.NoError(t, err)

	conn.Close()
}

func TestServerHeartBeatingTolerance(t *testing.T) {
	// Heart beat should not close connection exactly after not receiving message after cx
	//  it should add a pretty decent amount of time to counter network delay of other timing issues
	l, err := net.Listen("tcp", `127.0.0.1:0`)
	require.NoError(t, err)
	defer func() { l.Close() }()
	serv := Server{
		Addr:          l.Addr().String(),
		Authenticator: nil,
		QueueStorage:  nil,
		HeartBeat:     5 * time.Millisecond,
	}
	go serv.Serve(l)

	conn, err := net.Dial("tcp", l.Addr().String())
	require.NoError(t, err)
	defer conn.Close()

	client, err := stomp.Connect(conn,
		stomp.ConnOpt.HeartBeat(5*time.Millisecond, 5*time.Millisecond),
	)
	require.NoError(t, err)
	defer client.Disconnect()

	time.Sleep(serv.HeartBeat * 20) // let it go for some time to allow client and server to exchange some heart beat

	// Ensure the server has not closed his readChannel
	err = client.Send("/topic/whatever", "text/plain", []byte("hello"))
	require.NoError(t, err)
}

func TestServerSendToQueuesAndTopics(t *testing.T) {
	ch := make(chan bool, 2)
	println("number cpus:", runtime.NumCPU())

	addr := ":59092"

	l, err := net.Listen("tcp", addr)
	require.NoError(t, err)
	defer func() { l.Close() }()
	go Serve(l)

	// channel to communicate that the go routine has started
	started := make(chan bool)

	count := 100
	go runReceiver(t, ch, count, "/topic/test-1", addr, started)
	<-started
	go runReceiver(t, ch, count, "/topic/test-1", addr, started)
	<-started
	go runReceiver(t, ch, count, "/topic/test-2", addr, started)
	<-started
	go runReceiver(t, ch, count, "/topic/test-2", addr, started)
	<-started
	go runReceiver(t, ch, count, "/topic/test-1", addr, started)
	<-started
	go runReceiver(t, ch, count, "/queue/test-1", addr, started)
	<-started
	go runSender(t, ch, count, "/queue/test-1", addr, started)
	<-started
	go runSender(t, ch, count, "/queue/test-2", addr, started)
	<-started
	go runReceiver(t, ch, count, "/queue/test-2", addr, started)
	<-started
	go runSender(t, ch, count, "/topic/test-1", addr, started)
	<-started
	go runReceiver(t, ch, count, "/queue/test-3", addr, started)
	<-started
	go runSender(t, ch, count, "/queue/test-3", addr, started)
	<-started
	go runSender(t, ch, count, "/queue/test-4", addr, started)
	<-started
	go runSender(t, ch, count, "/topic/test-2", addr, started)
	<-started
	go runReceiver(t, ch, count, "/queue/test-4", addr, started)
	<-started

	for i := 0; i < 15; i++ {
		<-ch
	}
}

func runSender(t testing.TB, ch chan bool, count int, destination, addr string, started chan bool) {
	conn, err := net.Dial("tcp", "127.0.0.1"+addr)
	require.NoError(t, err)

	client, err := stomp.Connect(conn)
	require.NoError(t, err)

	started <- true

	for i := 0; i < count; i++ {
		client.Send(destination, "text/plain",
			[]byte(fmt.Sprintf("%s test message %d", destination, i)))
		//println("sent", i)
	}

	ch <- true
}

func runReceiver(t testing.TB, ch chan bool, count int, destination, addr string, started chan bool) {
	conn, err := net.Dial("tcp", "127.0.0.1"+addr)
	require.NoError(t, err)

	client, err := stomp.Connect(conn)
	require.NoError(t, err)

	sub, err := client.Subscribe(destination, stomp.AckAuto)
	require.NoError(t, err)
	require.NotNil(t, sub)

	started <- true

	for i := 0; i < count; i++ {
		msg := <-sub.C
		expectedText := fmt.Sprintf("%s test message %d", destination, i)
		require.Equal(t, []byte(expectedText), msg.Body)
		//println("received", i)
	}
	ch <- true
}
