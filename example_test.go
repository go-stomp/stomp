package stomp_test

import (
	"fmt"
	"github.com/guotie/stomp"
	"net"
	"time"
)

func ExampleConn_Send(c *stomp.Conn) error {
	// send with receipt and an optional header
	err := c.Send(
		"/queue/test-1",            // destination
		"text/plain",               // content-type
		[]byte("Message number 1"), // body
		stomp.NewHeader("expires", "2020-12-31 23:59:59"))
	if err != nil {
		return err
	}

	// send with no receipt and no optional headers
	err = c.Send("/queue/test-2", "application/xml",
		[]byte("<message>hello</message>"), nil)
	if err != nil {
		return err
	}

	return nil
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
	h := stomp.NewHeader(
		"login", "scott",
		"passcode", "tiger",
		"host", "stompserver",
		"accept-version", "1.1,1.2")
	doSomethingWith(h)
}

// Creates a STOMP frame.
func ExampleNewFrame() {
	/*
		Creates a STOMP frame that looks like the following:

			CONNECT
			login:scott
			passcode:tiger
			host:stompserver
			accept-version:1.1,1.2

			^@
	*/
	f := stomp.NewFrame("CONNECT",
		"login", "scott",
		"passcode", "tiger",
		"host", "stompserver",
		"accept-version", "1.1,1.2")
	doSomethingWith(f)
}

func doSomethingWith(f interface{}) {

}

func doAnotherThingWith(f interface{}, g interface{}) {

}

func ExampleSubscription() error {
	conn, err := stomp.Dial("tcp", "localhost:61613", stomp.Options{})
	if err != nil {
		return err
	}

	sub, err := conn.Subscribe("/queue/test-2", stomp.AckClient)
	if err != nil {
		return err
	}

	// receive 5 messages and then quit
	for i := 0; i < 5; i++ {
		msg := <-sub.C
		if msg.Err != nil {
			return msg.Err
		}

		doSomethingWith(msg)

		// acknowledge the message
		err = conn.Ack(msg)
		if err != nil {
			return err
		}
	}

	err = sub.Unsubscribe()
	if err != nil {
		return err
	}

	return conn.Disconnect()
}

func ExampleTransaction() error {
	conn, err := stomp.Dial("tcp", "localhost:61613", stomp.Options{})
	if err != nil {
		return err
	}
	defer conn.Disconnect()

	sub, err := conn.Subscribe("/queue/test-2", stomp.AckClient)
	if err != nil {
		return err
	}

	// receive 5 messages and then quit
	for i := 0; i < 5; i++ {
		msg := <-sub.C
		if msg.Err != nil {
			return msg.Err
		}

		tx := conn.Begin()

		doAnotherThingWith(msg, tx)

		tx.Send("/queue/another-one", "text/plain",
			[]byte(fmt.Sprintf("Message #%d", i)), nil)

		// acknowledge the message
		err = tx.Ack(msg)
		if err != nil {
			return err
		}

		err = tx.Commit()
		if err != nil {
			return err
		}
	}

	err = sub.Unsubscribe()
	if err != nil {
		return err
	}

	return nil
}

// Example of connecting to a STOMP server using an existing network connection.
func ExampleConnect() error {
	netConn, err := net.DialTimeout("tcp", "stomp.server.com:61613", 10*time.Second)
	if err != nil {
		return err
	}

	stompConn, err := stomp.Connect(netConn, stomp.Options{})
	if err != nil {
		return err
	}

	defer stompConn.Disconnect()

	doSomethingWith(stompConn)
	return nil
}

// Connect to a STOMP server using default options.
func ExampleDial_1() error {
	conn, err := stomp.Dial("tcp", "192.168.1.1:61613", stomp.Options{})
	if err != nil {
		return err
	}

	err = conn.Send(
		"/queue/test-1",           // destination
		"text/plain",              // content-type
		[]byte("Test message #1"), // body
		nil) // no headers
	if err != nil {
		return err
	}

	return conn.Disconnect()
}

// Connect to a STOMP server that requires authentication. In addition,
// we are only prepared to use STOMP protocol version 1.1 or 1.2, and
// the virtual host is named "dragon". In this example the STOMP
// server also accepts a non-standard header called 'nonce'.
func ExampleDial_2() error {
	conn, err := stomp.Dial("tcp", "192.168.1.1:61613", stomp.Options{
		Login:         "scott",
		Passcode:      "leopard",
		AcceptVersion: "1.1,1.2",
		Host:          "dragon",
		NonStandard:   stomp.NewHeader("nonce", "B256B26D320A"),
	})
	if err != nil {
		return err
	}

	err = conn.Send(
		"/queue/test-1",           // destination
		"text/plain",              // content-type
		[]byte("Test message #1"), // body
		nil) // no optional headers
	if err != nil {
		return err
	}

	return conn.Disconnect()
}
