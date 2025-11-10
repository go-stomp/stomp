package testutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFakeConn(t *testing.T) {
	//c.Skip("temporary")
	fc1, fc2 := NewFakeConn(t)

	one := []byte{1, 2, 3, 4}
	two := []byte{5, 6, 7, 8, 9, 10, 11, 12, 13}
	stop := make(chan struct{})

	go func() {
		defer func() {
			fc2.Close()
			close(stop)
		}()

		rx1 := make([]byte, 6)
		n, err := fc2.Read(rx1)
		require.Equal(t, 4, n)
		require.NoError(t, err)
		require.Equal(t, one, rx1[0:n])

		rx2 := make([]byte, 5)
		n, err = fc2.Read(rx2)
		require.Equal(t, 5, n)
		require.NoError(t, err)
		require.Equal(t, []byte{5, 6, 7, 8, 9}, rx2)

		rx3 := make([]byte, 10)
		n, err = fc2.Read(rx3)
		require.Equal(t, 4, n)
		require.NoError(t, err)
		require.Equal(t, []byte{10, 11, 12, 13}, rx3[0:n])
	}()

	require.Equal(t, t, fc1.T)
	require.Equal(t, t, fc2.T)

	n, err := fc1.Write(one)
	require.Equal(t, 4, n)
	require.NoError(t, err)

	n, err = fc1.Write(two)
	require.Len(t, two, n)
	require.NoError(t, err)

	<-stop
}
