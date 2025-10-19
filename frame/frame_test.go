package frame

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFrameNew(t *testing.T) {
	f := New("CCC")
	require.Equal(t, 0, f.Header.Len())
	require.Equal(t, "CCC", f.Command)

	f = New("DDDD", "abc", "def")
	require.Equal(t, 1, f.Header.Len())
	k, v := f.Header.GetAt(0)
	require.Equal(t, "abc", k)
	require.Equal(t, "def", v)
	require.Equal(t, "DDDD", f.Command)

	f = New("EEEEEEE", "abc", "def", "hij", "klm")
	require.Equal(t, "EEEEEEE", f.Command)
	require.Equal(t, 2, f.Header.Len())
	k, v = f.Header.GetAt(0)
	require.Equal(t, "abc", k)
	require.Equal(t, "def", v)
	k, v = f.Header.GetAt(1)
	require.Equal(t, "hij", k)
	require.Equal(t, "klm", v)
}

func TestFrameClone(t *testing.T) {
	f1 := &Frame{
		Command: "AAAA",
	}

	f2 := f1.Clone()
	require.Equal(t, f1.Command, f2.Command)
	require.Nil(t, f2.Header)
	require.Nil(t, f2.Body)

	f1.Header = NewHeader("aaa", "1", "bbb", "2", "ccc", "3")

	f2 = f1.Clone()
	require.Equal(t, f2.Header.Len(), f2.Header.Len())
	for i := 0; i < f1.Header.Len(); i++ {
		k1, v1 := f1.Header.GetAt(i)
		k2, v2 := f2.Header.GetAt(i)
		require.Equal(t, k1, k2)
		require.Equal(t, v1, v2)
	}

	f1.Body = []byte{1, 2, 3, 4, 5, 6, 5, 4, 77, 88, 99, 0xaa, 0xbb, 0xcc, 0xff}
	f2 = f1.Clone()
	require.Len(t, f2.Body, len(f1.Body))
	for i := 0; i < len(f1.Body); i++ {
		require.Equal(t, f1.Body[i], f2.Body[i])
	}
}
