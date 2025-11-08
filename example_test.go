package stomp_test

import (
	"fmt"
	"net"
	"time"

	"github.com/go-stomp/stomp/v3"
	"github.com/go-stomp/stomp/v3/frame"
)

func handle(_ error) {}

func ExampleConn_Send() {
	c, err := stomp.Dial("tcp", "localhost:61613")
	if err != nil {
		handle(err)
	}

	// send with receipt and an optional header
	err = c.Send(
		"/queue/test-1",            // destination
		"text/plain",               // content-type
		[]byte("Message number 1"), // body
		stomp.SendOpt.Receipt,
		stomp.SendOpt.Header("expires", "2049-12-31 23:59:59"))
	if err != nil {
		handle(err)
	}

	// send with no receipt and no optional headers
	err = c.Send("/queue/test-2", "application/xml",
		[]byte("<message>hello</message>"))
	if err != nil {
		handle(err)
	}

}

// Creates a new Header.
func ExampleNewHeader() {
	/*
		Creates a header that looks like the following:

			login:scott
			passcode:tiger
			host:stompserver
			accept-version:1.1,1.2
	*/
	h := frame.NewHeader(
		"login", "scott",
		"passcode", "tiger",
		"host", "stompserver",
		"accept-version", "1.1,1.2")
	doSomethingWith(h)
}

func doSomethingWith(f ...interface{}) {

}

func doAnotherThingWith(f interface{}, g interface{}) {

}

func ExampleConn_Subscribe() {
	conn, err := stomp.Dial("tcp", "localhost:61613")
	if err != nil {
		handle(err)
	}

	sub, err := conn.Subscribe("/queue/test-2", stomp.AckClient)
	if err != nil {
		handle(err)
	}

	// receive 5 messages and then quit
	for i := 0; i < 5; i++ {
		msg := <-sub.C
		if msg.Err != nil {
			handle(msg.Err)
		}

		doSomethingWith(msg)

		// acknowledge the message
		err = conn.Ack(msg)
		if err != nil {
			handle(err)
		}
	}

	err = sub.Unsubscribe()
	if err != nil {
		handle(err)
	}

	conn.Disconnect()
}

// Example of creating subscriptions with various options.
func ExampleConn_Subscribe_with_options() {
	c, err := stomp.Dial("tcp", "localhost:61613")
	if err != nil {
		handle(err)
	}

	// Subscribe to queue with automatic acknowledgement
	sub1, err := c.Subscribe("/queue/test-1", stomp.AckAuto)
	if err != nil {
		handle(err)
	}

	// Subscribe to queue with client acknowledgement and a custom header value
	sub2, err := c.Subscribe("/queue/test-2", stomp.AckClient,
		stomp.SubscribeOpt.Header("x-custom-header", "some-value"))
	if err != nil {
		handle(err)
	}

	doSomethingWith(sub1, sub2)
}

func ExampleTransaction() {
	conn, err := stomp.Dial("tcp", "localhost:61613")
	if err != nil {
		handle(err)
	}
	defer conn.Disconnect()

	sub, err := conn.Subscribe("/queue/test-2", stomp.AckClient)
	if err != nil {
		handle(err)
	}

	// receive 5 messages and then quit
	for i := 0; i < 5; i++ {
		msg := <-sub.C
		if msg.Err != nil {
			handle(msg.Err)
		}

		tx := conn.Begin()

		doAnotherThingWith(msg, tx)

		tx.Send("/queue/another-one", "text/plain",
			[]byte(fmt.Sprintf("Message #%d", i)), nil)

		// acknowledge the message
		err = tx.Ack(msg)
		if err != nil {
			handle(err)
		}

		err = tx.Commit()
		if err != nil {
			handle(err)
		}
	}

	err = sub.Unsubscribe()
	if err != nil {
		handle(err)
	}

}

// Example of connecting to a STOMP server using an existing network connection.
func ExampleConnect() {
	netConn, err := net.DialTimeout("tcp", "stomp.server.com:61613", 10*time.Second)
	if err != nil {
		handle(err)
	}

	stompConn, err := stomp.Connect(netConn)
	if err != nil {
		handle(err)
	}

	defer stompConn.Disconnect()

	doSomethingWith(stompConn)
}

// Connect to a STOMP server using default options.
func ExampleDial() {
	conn, err := stomp.Dial("tcp", "192.168.1.1:61613")
	if err != nil {
		handle(err)
	}

	err = conn.Send(
		"/queue/test-1",           // destination
		"text/plain",              // content-type
		[]byte("Test message #1")) // body
	if err != nil {
		handle(err)
	}

	conn.Disconnect()
}

// Connect to a STOMP server that requires authentication. In addition,
// we are only prepared to use STOMP protocol version 1.1 or 1.2, and
// the virtual host is named "dragon". In this example the STOMP
// server also accepts a non-standard header called 'nonce'.
func ExampleDial_with_options() {
	conn, err := stomp.Dial("tcp", "192.168.1.1:61613",
		stomp.ConnOpt.Login("scott", "leopard"),
		stomp.ConnOpt.AcceptVersion(stomp.V11),
		stomp.ConnOpt.AcceptVersion(stomp.V12),
		stomp.ConnOpt.Host("dragon"),
		stomp.ConnOpt.Header("nonce", "B256B26D320A"))
	if err != nil {
		handle(err)
	}

	err = conn.Send(
		"/queue/test-1",           // destination
		"text/plain",              // content-type
		[]byte("Test message #1")) // body
	if err != nil {
		handle(err)
	}

	conn.Disconnect()
}
