/*
A simple, stand-alone STOMP server.

TODO: graceful shutdown

TODO: UNIX daemon functionality

TODO: Windows service functionality (if possible?)

TODO: Logging options (syslog, windows event log)
*/
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/ingelectus/stomp/server"
)

// TODO: experimenting with ways to gracefully shutdown the server,
// at the moment it just dies ungracefully on SIGINT.

/*

func main() {
	// create a channel for listening for termination signals
	stopChannel := newStopChannel()

	for {
		select {
		case sig := <-stopChannel:
			log.Println("received signal:", sig)
			break
		}
	}

}
*/

var listenAddr = flag.String("addr", ":61613", "Listen address")
var helpFlag = flag.Bool("help", false, "Show this help text")

func main() {
	flag.Parse()
	if *helpFlag {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	l, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatalf("failed to listen: %s", err.Error())
	}
	defer func() { l.Close() }()

	log.Println("listening on", l.Addr().Network(), l.Addr().String())
	server.Serve(l)
}
