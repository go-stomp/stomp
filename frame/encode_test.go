package frame

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodeUnencodeValue(t *testing.T) {
	val, err := unencodeValue([]byte(`Contains\r\nNewLine and \c colon and \\ backslash`))
	require.NoError(t, err)
	require.Equal(t, "Contains\r\nNewLine and : colon and \\ backslash", val)
}
