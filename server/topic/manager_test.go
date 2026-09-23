package topic

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestManager(t *testing.T) {
	mgr := NewManager()

	t1 := mgr.Find("topic1")
	require.NotNil(t, t1)

	t2 := mgr.Find("topic2")
	require.NotNil(t, t2)

	require.Equal(t, t1, mgr.Find("topic1"))
}
