package frame

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFrameParseHeartBeat(t *testing.T) {
	testCases := []struct {
		Input             string
		ExpectedDuration1 time.Duration
		ExpectedDuration2 time.Duration
		ExpectError       bool
		ExpectedError     error
	}{
		{
			Input:             "0,0",
			ExpectedDuration1: 0,
			ExpectedDuration2: 0,
		},
		{
			Input:             "20000,60000",
			ExpectedDuration1: 20 * time.Second,
			ExpectedDuration2: time.Minute,
		},
		{
			Input:             "86400000,31536000000",
			ExpectedDuration1: 24 * time.Hour,
			ExpectedDuration2: 365 * 24 * time.Hour,
		},
		{
			Input:             "20r000,60000",
			ExpectedDuration1: 0,
			ExpectedDuration2: 0,
			ExpectedError:     ErrInvalidHeartBeat,
		},
		{
			Input:             "99999999999999999999,60000",
			ExpectedDuration1: 0,
			ExpectedDuration2: 0,
			ExpectedError:     ErrInvalidHeartBeat,
		},
		{
			Input:             "60000,99999999999999999999",
			ExpectedDuration1: 0,
			ExpectedDuration2: 0,
			ExpectedError:     ErrInvalidHeartBeat,
		},
		{
			Input:             "-60000,60000",
			ExpectedDuration1: 0,
			ExpectedDuration2: 0,
			ExpectedError:     ErrInvalidHeartBeat,
		},
		{
			Input:             "60000,-60000",
			ExpectedDuration1: 0,
			ExpectedDuration2: 0,
			ExpectedError:     ErrInvalidHeartBeat,
		},
	}

	for _, tc := range testCases {
		d1, d2, err := ParseHeartBeat(tc.Input)
		require.Equal(t, tc.ExpectedDuration1, d1)
		require.Equal(t, tc.ExpectedDuration2, d2)
		if tc.ExpectError || tc.ExpectedError != nil {
			require.Error(t, err)
			if tc.ExpectedError != nil {
				require.ErrorIs(t, err, tc.ExpectedError)
			}
		} else {
			require.NoError(t, err)
		}
	}
}
