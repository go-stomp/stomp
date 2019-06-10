package stomp_test

import (
	"testing"

	"github.com/ingelectus/stomp"
)

func TestSupportsNack(t *testing.T) {
	testCases := []struct {
		Version      stomp.Version
		SupportsNack bool
	}{
		{
			Version:      stomp.Version("1.0"),
			SupportsNack: false,
		},
		{
			Version:      stomp.Version("1.1"),
			SupportsNack: true,
		},
		{
			Version:      stomp.Version("1.2"),
			SupportsNack: true,
		},
		{
			Version:      stomp.Version("xxx"),
			SupportsNack: false,
		},
	}

	for _, testCase := range testCases {
		version := testCase.Version
		expected := testCase.SupportsNack
		actual := version.SupportsNack()
		if expected != actual {
			t.Errorf("Version %v: SupportsNack: expected %v, actual %v",
				version, expected, actual)
		}

	}

}

func TestCheckSupported(t *testing.T) {
	testCases := []struct {
		Version stomp.Version
		Err     error
	}{
		{
			Version: stomp.Version("1.0"),
			Err:     nil,
		},
		{
			Version: stomp.Version("1.1"),
			Err:     nil,
		},
		{
			Version: stomp.Version("1.2"),
			Err:     nil,
		},
		{
			Version: stomp.Version("2.2"),
			Err:     stomp.ErrUnsupportedVersion,
		},
	}

	for _, testCase := range testCases {
		version := testCase.Version
		expected := testCase.Err
		actual := version.CheckSupported()
		if expected != actual {
			t.Errorf("Version %v: CheckSupported: expected %v, actual %v",
				version, expected, actual)
		}

	}

}
