package client

import (
	"testing"

	"github.com/go-stomp/stomp/v3"
	"github.com/go-stomp/stomp/v3/frame"
	"github.com/stretchr/testify/require"
)

func TestFrameDetermineVersion_V10_Connect(t *testing.T) {
	f := frame.New(frame.CONNECT)
	version, err := determineVersion(f)
	require.NoError(t, err)
	require.Equal(t, stomp.V10, version)
}

func TestFrameDetermineVersion_V10_Stomp(t *testing.T) {
	// the "STOMP" command was introduced in V1.1, so it must
	// have an accept-version header
	f := frame.New(frame.STOMP)
	_, err := determineVersion(f)
	require.ErrorIs(t, err, missingHeader(frame.AcceptVersion))
}

func TestFrameDetermineVersion_V11_Connect(t *testing.T) {
	f := frame.New(frame.CONNECT)
	f.Header.Add(frame.AcceptVersion, "1.1")
	version, err := determineVersion(f)
	require.NoError(t, err)
	require.Equal(t, stomp.V11, version)
}

func TestFrameDetermineVersion_MultipleVersions(t *testing.T) {
	f := frame.New(frame.CONNECT)
	f.Header.Add(frame.AcceptVersion, "1.2,1.1,1.0,2.0")
	version, err := determineVersion(f)
	require.NoError(t, err)
	require.Equal(t, stomp.V12, version)
}

func TestFrameDetermineVersion_IncompatibleVersions(t *testing.T) {
	f := frame.New(frame.CONNECT)
	f.Header.Add(frame.AcceptVersion, "0.2,0.1,1.3,2.0")
	version, err := determineVersion(f)
	require.ErrorIs(t, err, unknownVersion)
	require.Equal(t, stomp.Version(""), version)
}

func TestFrameHeartBeat(t *testing.T) {
	f := frame.New(frame.CONNECT,
		frame.AcceptVersion, "1.2",
		frame.Host, "XX")

	// no heart-beat header means zero values
	x, y, err := getHeartBeat(f)
	require.NoError(t, err)
	require.Equal(t, 0, x)
	require.Equal(t, 0, y)

	f.Header.Add("heart-beat", "123,456")
	x, y, err = getHeartBeat(f)
	require.NoError(t, err)
	require.Equal(t, 123, x)
	require.Equal(t, 456, y)

	f.Header.Set(frame.HeartBeat, "invalid")
	x, y, err = getHeartBeat(f)
	require.Equal(t, 0, x)
	require.Equal(t, 0, y)
	require.ErrorIs(t, err, invalidHeartBeat)

	f.Header.Del(frame.HeartBeat)
	_, _, err = getHeartBeat(f)
	require.NoError(t, err)

	f.Command = frame.SEND
	_, _, err = getHeartBeat(f)
	require.ErrorIs(t, err, invalidOperationForFrame)
}
