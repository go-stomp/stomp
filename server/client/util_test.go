package client

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUtilAsMilliseconds(t *testing.T) {
	d := time.Duration(30) * time.Millisecond
	require.Equal(t, 30, asMilliseconds(d, math.MaxInt32))

	// approximately one year
	d = time.Duration(365) * time.Duration(24) * time.Hour
	require.Equal(t, math.MaxInt32, asMilliseconds(d, math.MaxInt32))

	d = time.Duration(365) * time.Duration(24) * time.Hour
	require.Equal(t, maxHeartBeat, asMilliseconds(d, maxHeartBeat))
}
