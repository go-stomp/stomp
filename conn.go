package stomp

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/guotie/stomp/frame"
)

// Default time span to add to read/write heart-beat timeouts
// to avoid premature disconnections due to network latency.
const DefaultReadHeartBeatError = 5 * time.Second

// Options for connecting to the STOMP server. Used with the
// stomp.Dial and stomp.Connect functions, both of which have examples.
type Options struct {
	// Login and passcode for authentication with the STOMP server.
	// If no authentication is required, leave blank.
	Login, Passcode string

	// Value for the "host" header entry when connecting to the
	// STOMP server. Leave blank for default value.
	Host string

	// Comma-separated list of acceptable STOMP versions.
	// Leave blank for default protocol negotiation, which is
	// the recommended setting.
	AcceptVersion string

	// Value to pass in the "heart-beat" header entry when connecting
	// to the STOMP server. Format is two non-negative integer values
	// separated by a comma. Leave blank for default heart-beat negotiation,
	// which is the recommended setting.
	HeartBeat string

	// As per the STOMP specification, we should expect some discrepancy
	// in incoming heartbeats. Add this time duration to the read timeout.
	// If this value is zero, it will be set to DefaultReadHeartBeatError.
	// If you really want to have zero error (eg for testing), set to a
	// negative value.
	ReadHeartBeatError time.Duration

	// Other header entries for STOMP servers that accept non-standard
	// header entries in the CONNECT frame.
	NonStandard *Header
}

type connChan struct {
	sync.Mutex
	channels map[string]chan *Frame
}

// A Conn is a connection to a STOMP server. Create a Conn using either
// the Dial or Connect function.
type Conn struct {
	conn         io.ReadWriteCloser
	readCh       chan *Frame
	writeCh      chan writeRequest
	version      Version
	session      string
	server       string
	readTimeout  time.Duration
	writeTimeout time.Duration
	closed       bool

	// 2015-08-19 guotie
	// 读写分离
	channels connChan
}

type writeRequest struct {
	Frame *Frame      // frame to send
	C     chan *Frame // response channel
}

// Dial creates a network connection to a STOMP server and performs
// the STOMP connect protocol sequence. The network endpoint of the
// STOMP server is specified by network and addr. STOMP protocol
// options can be specified in opts.
func Dial(network, addr string, opts Options) (*Conn, error) {
	c, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	log.Printf("dial %s success.\n", addr)

	if opts.Host == "" {
		host, _, err := net.SplitHostPort(c.RemoteAddr().String())
		if err != nil {
			c.Close()
			return nil, err
		}
		opts.Host = host
	}

	return Connect(c, opts)
}

// Connect creates a STOMP connection and performs the STOMP connect
// protocol sequence. The connection to the STOMP server has already
// been created by the program. The opts parameter provides the
// opportunity to specify STOMP protocol options.
func Connect(conn io.ReadWriteCloser, opts Options) (*Conn, error) {
	reader := NewReader(conn)
	writer := NewWriter(conn)

	// set default values
	if opts.AcceptVersion == "" {
		opts.AcceptVersion = "1.0,1.1,1.2"
	}
	if opts.HeartBeat == "" {
		opts.HeartBeat = "60000,60000"
	}
	if opts.Host == "" {
		// Attempt to get host from net.Conn object if possible
		if connection, ok := conn.(net.Conn); ok {
			host, _, err := net.SplitHostPort(connection.RemoteAddr().String())
			if err == nil {
				opts.Host = host
			}
		}

		// If host is still blank, use default
		if opts.Host == "" {
			opts.Host = "default"
		}
	}

	connectFrame := NewFrame(frame.CONNECT,
		frame.Host, opts.Host,
		frame.AcceptVersion, opts.AcceptVersion,
		frame.HeartBeat, opts.HeartBeat)
	if opts.Login != "" || opts.Passcode != "" {
		connectFrame.Set(frame.Login, opts.Login)
		connectFrame.Set(frame.Passcode, opts.Passcode)
	}
	if opts.NonStandard != nil {
		for i := 0; i < opts.NonStandard.Len(); i++ {
			key, value := opts.NonStandard.GetAt(i)
			connectFrame.Add(key, value)
		}
	}

	err := writer.Write(connectFrame)
	if err != nil {
		return nil, err
	}

	response, err := reader.Read()
	if err != nil {
		return nil, err
	}

	if response.Command != frame.CONNECTED {
		return nil, newError(response)
	}

	c := &Conn{
		conn:    conn,
		readCh:  make(chan *Frame, 8),
		writeCh: make(chan writeRequest, 8),
		server:  response.Get(frame.Server),
		session: response.Get(frame.Session),

		channels: connChan{channels: make(map[string]chan *Frame)},
	}

	if version := response.Get(frame.Version); version != "" {
		switch Version(version) {
		case V10, V11, V12:
			c.version = Version(version)
		default:
			return nil, Error{Message: "unsupported version", Frame: response}
		}
	} else {
		c.version = V10
	}

	if heartBeat, ok := response.Contains(frame.HeartBeat); ok {
		readTimeout, writeTimeout, err := frame.ParseHeartBeat(heartBeat)
		if err != nil {
			return nil, Error{
				Message: err.Error(),
				Frame:   response,
			}
		}
		rtError := opts.ReadHeartBeatError
		if rtError < 0 {
			rtError = 0
		} else if rtError == 0 {
			rtError = DefaultReadHeartBeatError
		}

		c.readTimeout = readTimeout + rtError
		c.writeTimeout = writeTimeout / 2
	}

	// TODO(jpj): make any non-standard headers in the CONNECTED
	// frame available.

	go readLoop(c, reader)
	go processReadLoop(c)
	go processLoop(c, writer)

	return c, nil
}

// Version returns the version of the STOMP protocol that
// is being used to communicate with the STOMP server. This
// version is negotiated with the server during the connect sequence.
func (c *Conn) Version() Version {
	return c.version
}

// Session returns the session identifier, which can be
// returned by the STOMP server during the connect sequence.
// If the STOMP server does not return a session header entry,
// this value will be a blank string.
func (c *Conn) Session() string {
	return c.session
}

// Server returns the STOMP server identification, which can
// be returned by the STOMP server during the connect sequence.
// If the STOMP server does not return a server header entry,
// this value will be a blank string.
func (c *Conn) Server() string {
	return c.server
}

// readLoop is a goroutine that reads frames from the
// reader and places them onto a channel for processing
// by the processLoop goroutine
func readLoop(c *Conn, reader *Reader) {
	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
			return
		}
	}()

	for {
		f, err := reader.Read()
		if err != nil {
			log.Printf("read failed: %s, exit readLoop\n", err.Error())
			close(c.readCh)
			return
		}
		c.readCh <- f
	}
}

// close connection
func closeConn(c *Conn, closeReadCh bool) {
	defer func() {
		if err := recover(); err != nil {
			// 记录到日志中
			log.Printf("closeConn: %v\n", err)
		}
	}()

	if c.closed {
		return
	}

	if closeReadCh {
		close(c.readCh)
	}
	close(c.writeCh)
	c.conn.Close()
	c.closed = true
}

// 避免发送到channel的阻塞操作，先把map copy一份
func copyChannels(cc connChan) *connChan {
	var m = connChan{}

	cc.Lock()
	for key, v := range cc.channels {
		m.channels[key] = v
	}
	cc.Unlock()
	return &m
}

func processReadLoop(c *Conn) {
	var readTimeoutChannel <-chan time.Time
	var readTimer *time.Timer

	defer func() {
		if err := recover(); err != nil {
			// 记录到日志中
			log.Printf("processReadLoop: %v\n", err)
		}
	}()

	for {
		if c.readTimeout > 0 && readTimer == nil {
			readTimer := time.NewTimer(c.readTimeout)
			readTimeoutChannel = readTimer.C
		}
		select {
		case <-readTimeoutChannel:
			// read timeout, close the connection
			err := newErrorMessage("read timeout")
			sendError(c, err)
			log.Println("read channel Timeout, exit processLoop.")
			closeConn(c, true)
			return
		case f, ok := <-c.readCh:
			// stop the read timer
			if readTimer != nil {
				readTimer.Stop()
				readTimer = nil
				readTimeoutChannel = nil
			}

			if !ok {
				err := newErrorMessage("connection closed")
				sendError(c, err)
				log.Println("readCh closed, exit processLoop.")
				closeConn(c, false)
				return
			}

			if f == nil {
				// heart-beat received
				continue
			}

			switch f.Command {
			case frame.RECEIPT:
				if id, ok := f.Contains(frame.ReceiptId); ok {
					c.channels.Lock()
					ch, ok := c.channels.channels[id]
					if ok {
						delete(c.channels.channels, id)
					}
					c.channels.Unlock()
					if ok {
						ch <- f
						close(ch)
					}

				} else {
					err := &Error{Message: "missing receipt-id", Frame: f}
					sendError(c, err)
					closeConn(c, true)
					//return
					continue
				}

			case frame.ERROR:
				log.Println("received ERROR; Closing underlying connection, exit processLoop")
				c.channels.Lock()
				nc := copyChannels(c.channels)
				c.channels.Unlock()

				for _, ch := range nc.channels {
					ch <- f
					close(ch)
				}

				closeConn(c, true)

				return

			case frame.MESSAGE:
				if id, ok := f.Contains(frame.Subscription); ok {
					c.channels.Lock()
					ch1, ok := c.channels.channels[id]
					c.channels.Unlock()
					if ok {
						ch1 <- f
					} else {
						log.Println("ignored MESSAGE for subscription", id)
					}
				}
			}
		}
	}
}

// processLoop is a goroutine that handles io with
// the server.
func processLoop(c *Conn, writer *Writer) {
	//channels := make(map[string]chan *Frame)
	//channels := c.channels

	//var readTimeoutChannel <-chan time.Time
	//var readTimer *time.Timer
	var writeTimeoutChannel <-chan time.Time
	var writeTimer *time.Timer

	defer func() {
		if err := recover(); err != nil {
			// 记录到日志中
			log.Printf("processLoop: %v\n", err)
		}
	}()

	for {
		/*
			if c.readTimeout > 0 && readTimer == nil {
				readTimer := time.NewTimer(c.readTimeout)
				readTimeoutChannel = readTimer.C
			}
		*/
		if c.writeTimeout > 0 && writeTimer == nil {
			writeTimer := time.NewTimer(c.writeTimeout)
			writeTimeoutChannel = writeTimer.C
		}

		select {
		/*
			case <-readTimeoutChannel:
				// read timeout, close the connection
				err := newErrorMessage("read timeout")
				sendError(c, err)
				log.Println("read channel Timeout, exit processLoop.")
				closeConn(c, true)
				return
		*/
		case <-writeTimeoutChannel:
			// write timeout, send a heart-beat frame
			err := writer.Write(nil)
			if err != nil {
				sendError(c, err)
				log.Printf("write heart-beat to server failed: %s, exit processLoop.\n", err.Error())
				closeConn(c, true)
				return
			}
			writeTimer = nil
			writeTimeoutChannel = nil
			/*
				case f, ok := <-c.readCh:
					// stop the read timer
					if readTimer != nil {
						readTimer.Stop()
						readTimer = nil
						readTimeoutChannel = nil
					}

					if !ok {
						err := newErrorMessage("connection closed")
						sendError(channels, err)
						log.Println("readCh closed, exit processLoop.")
						closeConn(c, false)
						return
					}

					if f == nil {
						// heart-beat received
						continue
					}

					switch f.Command {
					case frame.RECEIPT:
						if id, ok := f.Contains(frame.ReceiptId); ok {
							if ch, ok := channels[id]; ok {
								ch <- f
								delete(channels, id)
								close(ch)
							}

						} else {
							err := &Error{Message: "missing receipt-id", Frame: f}
							sendError(channels, err)
							closeConn(c, true)
							//return
							continue
						}

					case frame.ERROR:
						log.Println("received ERROR; Closing underlying connection, exit processLoop")
						for _, ch := range channels {
							ch <- f
							close(ch)
						}

						closeConn(c, true)

						return

					case frame.MESSAGE:
						if id, ok := f.Contains(frame.Subscription); ok {
							if ch, ok := channels[id]; ok {
								ch <- f
							} else {
								log.Println("ignored MESSAGE for subscription", id)
							}
						}
					}
			*/
		case req, ok := <-c.writeCh:
			// stop the write timeout
			if writeTimer != nil {
				writeTimer.Stop()
				writeTimer = nil
				writeTimeoutChannel = nil
			}

			if !ok {
				sendError(c, errors.New("write channel closed"))
				log.Println("write channle closed, exit processLoop.")
				closeConn(c, true)
				return
			}
			if req.C != nil {
				if receipt, ok := req.Frame.Contains(frame.Receipt); ok {
					// remember the channel for this receipt
					c.channels.Lock()
					c.channels.channels[receipt] = req.C
					c.channels.Unlock()
				}
			}

			switch req.Frame.Command {
			case frame.SUBSCRIBE:
				id, _ := req.Frame.Contains(frame.Id)
				c.channels.Lock()
				c.channels.channels[id] = req.C
				c.channels.Unlock()
			case frame.UNSUBSCRIBE:
				id, _ := req.Frame.Contains(frame.Id)
				// is this trying to be too clever -- add a receipt
				// header so that when the server responds with a
				// RECEIPT frame, the corresponding channel will be closed
				req.Frame.Set(frame.Receipt, id)
			}

			// frame to send
			// 2015-08-19 gutie
			// 这里可能会阻塞，导致整个select阻塞，无法从队列中读取消息
			err := writer.Write(req.Frame)
			if err != nil {
				sendError(c, err)
				log.Printf("send frame to server failed: %s, exit processLoop.\n", err.Error())
				closeConn(c, true)
				return
			}

		}
	}
}

// Send an error to all receipt channels.
//func sendError(m map[string]chan *Frame, err error) {
func sendError(c *Conn, err error) {
	var m = make(map[string]chan *Frame)
	log.Println("sendError:", err)

	c.channels.Lock()
	for key, ch := range c.channels.channels {
		m[key] = ch
	}
	c.channels.Unlock()

	frame := NewFrame(frame.ERROR, frame.Message, err.Error())
	for _, ch := range m {
		ch <- frame
	}
}

// Disconnect will disconnect from the STOMP server. This function
// follows the STOMP standard's recommended protocol for graceful
// disconnection: it sends a DISCONNECT frame with a receipt header
// element. Once the RECEIPT frame has been received, the connection
// with the STOMP server is closed and any further attempt to write
// to the server will fail.
func (c *Conn) Disconnect() error {
	if c.closed {
		return nil
	}

	ch := make(chan *Frame)
	c.writeCh <- writeRequest{
		Frame: NewFrame(frame.DISCONNECT, frame.Receipt, allocateId()),
		C:     ch,
	}

	response := <-ch
	if response.Command != frame.RECEIPT {
		return newError(response)
	}

	c.closed = true
	return c.conn.Close()
}

func (c *Conn) DisconnectMust() error {
	if c.closed {
		return nil
	}

	// just close readCh and writeCh
	// close(c.readCh)
	close(c.writeCh)

	c.closed = true
	return c.conn.Close()
}

// Send sends a message to the STOMP server, which in turn sends the message to the specified destination.
// This method returns without confirming that the STOMP server has received the message. If the STOMP server
// does fail to receive the message for any reason, the connection will close.
//
// The content type should be specified, according to the STOMP specification, but if contentType is an empty
// string, the message will be delivered without a content type header entry. The body array contains the
// message body, and its content should be consistent with the specified content type.
//
// The message can contain optional, user-defined header entries in userDefined. If there are no optional header
// entries, then set userDefined to nil.
func (c *Conn) Send(destination, contentType string, body []byte, userDefined *Header) error {
	if c.closed {
		return newErrorMessage("Underlying connection closed.")
	}

	f := createSendFrame(destination, contentType, body, userDefined)
	f.Del(frame.Transaction)
	err := c.sendFrame(f)
	return err
}

// Send sends a message to the STOMP server, which in turn sends the message to the specified destination.
// This method does not return until the STOMP server has confirmed receipt of the message.
//
// The content type should be specified, according to the STOMP specification, but if contentType is an empty
// string, the message will be delivered without a content type header entry. The body array contains the
// message body, and its content should be consistent with the specified content type.
//
// The message can contain optional, user-defined header entries in userDefined. If there are no optional header
// entries, then set userDefined to nil.
func (c *Conn) SendWithReceipt(destination, contentType string, body []byte, userDefined *Header) error {
	if c.closed {
		return newErrorMessage("Underlying connection closed.")
	}

	f := createSendFrame(destination, contentType, body, userDefined)
	f.Del(frame.Transaction)
	return c.sendFrameWithReceipt(f)
}

func createSendFrame(destination, contentType string, body []byte, userDefined *Header) *Frame {
	f := &Frame{
		Command: frame.SEND,
		Body:    body,
	}
	if userDefined == nil {
		f.Header = NewHeader()
	} else {
		f.Header = userDefined.Clone()
		f.Header.Del(frame.Receipt)
	}

	f.Header.Set(frame.Destination, destination)

	if contentType == "" {
		// no content type specified
		f.Header.Del(frame.ContentType)
	} else {
		f.Header.Set(frame.ContentType, contentType)
	}

	f.Header.Set(frame.ContentLength, strconv.Itoa(len(body)))
	return f
}

func (c *Conn) sendFrame(f *Frame) error {
	var err error
	// send frame
	defer func() error {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
		return err
	}()

	request := writeRequest{Frame: f}
	c.writeCh <- request

	return err
}

func (c *Conn) sendFrameWithReceipt(f *Frame) error {
	var err error
	defer func() error {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
		return err
	}()

	receipt := allocateId()
	f.Set(frame.Receipt, receipt)

	request := writeRequest{Frame: f}

	request.C = make(chan *Frame)
	c.writeCh <- request
	response := <-request.C
	if response.Command != frame.RECEIPT {
		return newError(response)
	}
	// TODO(jpj) Check receipt id

	return nil
}

// Subscribe creates a subscription on the STOMP server.
// The subscription has a destination, and messages sent to that destination
// will be received by this subscription. A subscription has a channel
// on which the calling program can receive messages.
func (c *Conn) Subscribe(destination string, ack AckMode) (*Subscription, error) {
	return c.SubscribeWithHeaders(destination, ack, NewHeader())
}

func (c *Conn) SubscribeWithHeaders(destination string, ack AckMode, userDefined *Header) (*Subscription, error) {
	ch := make(chan *Frame)
	headers := userDefined.Clone()

	// 2015-08-19
	// 在下面的go sub.readLoop(ch)发生panic，不得解
	defer func() error {
		var err error
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
		return err
	}()

	if _, ok := headers.Contains(frame.Id); !ok {
		headers.Add(frame.Id, allocateId())
	}

	headers.Add(frame.Destination, destination)
	headers.Add(frame.Ack, ack.String())

	f := NewFrame(frame.SUBSCRIBE)
	f.Header = headers

	request := writeRequest{Frame: f, C: ch}

	sub := &Subscription{
		id:          headers.Get(frame.Id),
		destination: destination,
		conn:        c,
		ackMode:     ack,
		C:           make(chan *Message, 16),
	}
	go sub.readLoop(ch)

	c.writeCh <- request
	return sub, nil
}

// Ack acknowledges a message received from the STOMP server.
// If the message was received on a subscription with AckMode == AckAuto,
// then no operation is performed.
func (c *Conn) Ack(m *Message) error {
	f, err := c.createAckNackFrame(m, true)
	if err != nil {
		return err
	}

	if f != nil {
		c.sendFrame(f)
	}
	return nil
}

func (c *Conn) Nack(m *Message) error {
	f, err := c.createAckNackFrame(m, false)
	if err != nil {
		return err
	}

	if f != nil {
		c.sendFrame(f)
	}
	return nil
}

// Begin is used to start a transaction. Transactions apply to sending
// and acknowledging. Any messages sent or acknowledged during a transaction
// will be processed atomically by the STOMP server based on the transaction.
func (c *Conn) Begin() *Transaction {
	id := allocateId()
	f := NewFrame(frame.BEGIN, frame.Transaction, id)
	c.sendFrame(f)
	return &Transaction{id: id, conn: c}
}

// Create an ACK or NACK frame. Complicated by version incompatibilities.
func (c *Conn) createAckNackFrame(msg *Message, ack bool) (*Frame, error) {
	if !ack && !c.version.SupportsNack() {
		return nil, nackNotSupported
	}

	if msg.Header == nil || msg.Subscription == nil || msg.Conn == nil {
		return nil, notReceivedMessage
	}

	if msg.Subscription.AckMode() == AckAuto {
		if ack {
			// not much point sending an ACK to an auto subscription
			return nil, nil
		} else {
			// sending a NACK for an ack:auto subscription makes no
			// sense
			return nil, cannotNackAutoSub
		}
	}

	var f *Frame
	if ack {
		f = NewFrame(frame.ACK)
	} else {
		f = NewFrame(frame.NACK)
	}

	switch c.version {
	case V10, V11:
		f.Header.Add(frame.Subscription, msg.Subscription.Id())
		if messageId, ok := msg.Header.Contains(frame.MessageId); ok {
			f.Header.Add(frame.MessageId, messageId)
		} else {
			return nil, missingHeader(frame.MessageId)
		}
	case V12:
		if ack, ok := msg.Header.Contains(frame.Ack); ok {
			f.Header.Add(frame.Id, ack)
		} else {
			return nil, missingHeader(frame.Ack)
		}
	}

	return f, nil
}
