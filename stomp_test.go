package stomp

import (
	//"gopkg.in/check.v1"
	"testing"
)

// Runs all gocheck tests in this package.
// See other *_test.go files for gocheck tests.
func TestStomp(t *testing.T) {
	check.TestingT(t)
}

type StompSuite struct{}

var _ = check.Suite(&StompSuite{})
