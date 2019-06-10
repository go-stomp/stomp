package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ingelectus/stomp"
)

const defaultPort = ":61613"

var serverAddr = flag.String("server", "localhost:61613", "STOMP server endpoint")
var messageCount = flag.Int("count", 10, "Number of messages to send/receive")
var queueName = flag.String("queue", "/queue/client_test", "Destination queue")
var helpFlag = flag.Bool("help", false, "Print help text")
var stop = make(chan bool)

// these are the default options that work with RabbitMQ
var options []func(*stomp.Conn) error = []func(*stomp.Conn) error{
	stomp.ConnOpt.Login("guest", "guest"),
	stomp.ConnOpt.Host("/"),
}

func main() {
	flag.Parse()
	if *helpFlag {
		fmt.Fprintf(os.Stderr, "Usage of %s\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	subscribed := make(chan bool)
	go recvMessages(subscribed)

	// wait until we know the receiver has subscribed
	<-subscribed

	go sendMessages()

	<-stop
	<-stop
}

func sendMessages() {
	defer func() {
		stop <- true
	}()

	conn, err := stomp.Dial("tcp", *serverAddr, options...)
	if err != nil {
		println("cannot connect to server", err.Error())
		return
	}

	for i := 1; i <= *messageCount; i++ {
		text := fmt.Sprintf("Message #%d", i)
		err = conn.Send(*queueName, "text/plain",
			[]byte(text), nil)
		if err != nil {
			println("failed to send to server", err)
			return
		}
	}
	println("sender finished")
}

func recvMessages(subscribed chan bool) {
	defer func() {
		stop <- true
	}()

	conn, err := stomp.Dial("tcp", *serverAddr, options...)

	if err != nil {
		println("cannot connect to server", err.Error())
		return
	}

	sub, err := conn.Subscribe(*queueName, stomp.AckAuto)
	if err != nil {
		println("cannot subscribe to", *queueName, err.Error())
		return
	}
	close(subscribed)

	for i := 1; i <= *messageCount; i++ {
		msg := <-sub.C
		expectedText := fmt.Sprintf("Message #%d", i)
		actualText := string(msg.Body)
		if expectedText != actualText {
			println("Expected:", expectedText)
			println("Actual:", actualText)
		}
	}
	println("receiver finished")

}
