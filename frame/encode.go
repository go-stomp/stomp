package frame

import (
	"bytes"
	"strings"
)

var (
	replacerForEncodeValue = strings.NewReplacer(
		"\\", "\\\\",
		"\r", "\\r",
		"\n", "\\n",
		":", "\\c",
	)
	replacerForUnencodeValue = strings.NewReplacer(
		"\\r", "\r",
		"\\n", "\n",
		"\\c", ":",
		"\\\\", "\\",
	)
)


type appendSliceWriter []byte

// Write writes to the buffer to satisfy io.Writer.
func (w *appendSliceWriter) Write(p []byte) (int, error) {
	*w = append(*w, p...)
	return len(p), nil
}

// WriteString writes to the buffer without string->[]byte->string allocations.
func (w *appendSliceWriter) WriteString(s string) (int, error) {
	*w = append(*w, s...)
	return len(s), nil
}

// Encodes a header value using STOMP value encoding
func encodeValue(s string) []byte {
	buf := make(appendSliceWriter, 0, len(s))
	replacerForEncodeValue.WriteString(&buf, s)
	return buf
}

// Unencodes a header value using STOMP value encoding
// TODO: return error if invalid sequences found (eg "\t")
func unencodeValue(b []byte) (string, error) {
	s := replacerForUnencodeValue.Replace(string(b))
	return s, nil
}
