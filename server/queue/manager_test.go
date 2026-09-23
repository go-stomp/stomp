package queue

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestManager(t *testing.T) {
	mgr := NewManager(NewMemoryQueueStorage())

	q1 := mgr.Find("/queue/1")
	require.NotNil(t, q1)

	q2 := mgr.Find("/queue/2")
	require.NotNil(t, q2)

	require.Equal(t, q1, mgr.Find("/queue/1"))
}
