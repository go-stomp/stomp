package stomp

// A Frame represents a STOMP frame. A frame consists of a command
// followed by a collection of header elements, and then an optional
// body.
//
// Users of this package will not normally need to make use of the Frame
// type directly. It is a lower level type useful for implementing
// STOMP protocol handlers.
type Frame struct {
	Command string
	*Header
	Body []byte
}

// NewFrame creates a new STOMP frame with the specified command and headers.
// The headers should contain an even number of entries. Each even index is
// the header name, and the odd indexes are the assocated header values.
func NewFrame(command string, headers ...string) *Frame {
	f := &Frame{Command: command, Header: &Header{}}
	for index := 0; index < len(headers); index += 2 {
		f.Add(headers[index], headers[index+1])
	}
	return f
}

// Clone creates a deep copy of the frame and its header. The cloned
// frame shares the body with the original frame.
func (f *Frame) Clone() *Frame {
	return &Frame{Command: f.Command, Header: f.Header.Clone(), Body: f.Body}
}

func (f *Frame) verifyConnect(version Version, isStomp bool) error {
	switch version {
	case V10:
		if isStomp {

		}
	case V11:
	case V12:
	}
	return nil
}

func (f *Frame) verifyMandatory(keys ...string) error {
	for _, key := range keys {
		if _, ok := f.Header.index(key); !ok {
			return &Error{
				Message: "missing header: " + key,
				Frame:   f,
			}
		}
	}
	return nil
}

func (f *Frame) HeaderBytes() (buf []byte) {
	buf = append(buf, []byte(f.Command)...)
	buf = append(buf, newlineSlice...)
	if f.Header != nil {
		for i := 0; i < f.Header.Len(); i++ {
			key, value := f.Header.GetAt(i)
			buf = append(buf, []byte(key)...)
			buf = append(buf, colonSlice...)
			buf = append(buf, []byte(value)...)
			buf = append(buf, newlineSlice...)
		}
	}
	buf = append(buf, newlineSlice...)
	return
}
