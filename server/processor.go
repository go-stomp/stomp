package server

import (
	"net"
	"strings"
	"time"

	"github.com/go-stomp/stomp/v3"
	"github.com/go-stomp/stomp/v3/frame"
	"github.com/go-stomp/stomp/v3/server/client"
	"github.com/go-stomp/stomp/v3/server/queue"
	"github.com/go-stomp/stomp/v3/server/topic"
)

type requestProcessor struct {
	server *Server
	ch     chan client.Request
	tm     *topic.Manager
	qm     *queue.Manager
}

func newRequestProcessor(server *Server) *requestProcessor {
	proc := &requestProcessor{
		server: server,
		ch:     make(chan client.Request, 128),
		tm:     topic.NewManager(),
	}

	if server.QueueStorage == nil {
		proc.qm = queue.NewManager(queue.NewMemoryQueueStorage())
	} else {
		proc.qm = queue.NewManager(server.QueueStorage)
	}

	return proc
}

func (proc *requestProcessor) Serve(l net.Listener) error {
	go proc.Listen(l)

	for {
		r := <-proc.ch
		switch r.Op {
		case client.SubscribeOp:
			if isQueueDestination(r.Sub.Destination()) {
				queue := proc.qm.Find(r.Sub.Destination())
				if err := queue.Subscribe(r.Sub); err != nil {
					proc.server.Log.Errorf("failed to subscribe %s: %v", r.Sub.Destination(), err)
				}
			} else {
				topic := proc.tm.Find(r.Sub.Destination())
				topic.Subscribe(r.Sub)
			}

		case client.UnsubscribeOp:
			if isQueueDestination(r.Sub.Destination()) {
				queue := proc.qm.Find(r.Sub.Destination())
				// todo error handling
				queue.Unsubscribe(r.Sub)
			} else {
				topic := proc.tm.Find(r.Sub.Destination())
				topic.Unsubscribe(r.Sub)
			}

		case client.EnqueueOp:
			destination, ok := r.Frame.Header.Contains(frame.Destination)
			if !ok {
				// should not happen, already checked in lower layer
				panic("missing destination")
			}

			if isQueueDestination(destination) {
				queue := proc.qm.Find(destination)
				if err := queue.Enqueue(r.Frame); err != nil {
					proc.server.Log.Errorf("failed to enqueue to %s: %v", destination, err)
				}
			} else {
				topic := proc.tm.Find(destination)
				topic.Enqueue(r.Frame)
			}

		case client.RequeueOp:
			destination, ok := r.Frame.Header.Contains(frame.Destination)
			if !ok {
				// should not happen, already checked in lower layer
				panic("missing destination")
			}

			// only requeue to queues, should never happen for topics
			if isQueueDestination(destination) {
				queue := proc.qm.Find(destination)
				if err := queue.Requeue(r.Frame); err != nil {
					proc.server.Log.Errorf("failed to requeue to %s: %v", destination, err)
				}
			}
		}
	}
}

func isQueueDestination(dest string) bool {
	return strings.HasPrefix(dest, QueuePrefix)
}

func (proc *requestProcessor) Listen(l net.Listener) {
	config := newConfig(proc.server)
	timeout := time.Duration(0) // how long to sleep on accept failure
	for {
		rw, err := l.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				if timeout == 0 {
					timeout = 5 * time.Millisecond
				} else {
					timeout *= 2
				}
				if max := 5 * time.Second; timeout > max {
					timeout = max
				}
				proc.server.Log.Infof("stomp: Accept error: %v; retrying in %v", err, timeout)
				time.Sleep(timeout)
				continue
			}
			return
		}
		timeout = 0
		// TODO: need to pass Server to connection so it has access to
		// configuration parameters.
		_ = client.NewConn(config, rw, proc.ch)
	}
}

type config struct {
	server *Server
}

func newConfig(s *Server) *config {
	return &config{server: s}
}

func (c *config) HeartBeat() time.Duration {
	if c.server.HeartBeat == time.Duration(0) {
		return DefaultHeartBeat
	}
	return c.server.HeartBeat
}

func (c *config) Authenticate(login, passcode string) bool {
	if c.server.Authenticator != nil {
		return c.server.Authenticator.Authenticate(login, passcode)
	}

	// no authentication defined
	return true
}

func (c *config) Logger() stomp.Logger {
	return c.server.Log
}
