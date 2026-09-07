package client

const (
	notConnected             = errorMessage("expected CONNECT or STOMP frame")
	unexpectedCommand        = errorMessage("unexpected frame command")
	unknownCommand           = errorMessage("unknown command")
	receiptInConnect         = errorMessage("receipt header prohibited in CONNECT or STOMP frame")
	authenticationFailed     = errorMessage("authentication failed")
	txAlreadyInProgress      = errorMessage("transaction already in progress")
	txUnknown                = errorMessage("unknown transaction")
	unsupportedVersion       = errorMessage("unsupported version")
	subscriptionExists       = errorMessage("subscription already exists")
	subscriptionNotFound     = errorMessage("subscription not found")
	invalidFrameFormat       = errorMessage("invalid frame format")
	invalidCommand           = errorMessage("invalid command")
	unknownVersion           = errorMessage("incompatible version")
	notConnectFrame          = errorMessage("operation valid for STOMP and CONNECT frames only")
	invalidHeartBeat         = errorMessage("invalid format for heart-beat")
	invalidOperationForFrame = errorMessage("invalid operation for frame")
	exceededMaxFrameSize     = errorMessage("exceeded max frame size")
	invalidHeaderValue       = errorMessage("invalid header value")
)

type errorMessage string

func (e errorMessage) Error() string {
	return string(e)
}

func missingHeader(name string) errorMessage {
	return errorMessage("missing header: " + name)
}
