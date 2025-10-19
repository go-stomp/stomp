package client

import (
	"testing"

	"github.com/go-stomp/stomp/v3/frame"
	"github.com/stretchr/testify/require"
)

func TestTxStoreDoubleBegin(t *testing.T) {
	txs := txStore{}

	err := txs.Begin("tx1")
	require.NoError(t, err)

	err = txs.Begin("tx1")
	require.ErrorIs(t, err, txAlreadyInProgress)
}

func TestTxStoreSuccessfulTx(t *testing.T) {
	txs := txStore{}

	err := txs.Begin("tx1")
	require.NoError(t, err)

	err = txs.Begin("tx2")
	require.NoError(t, err)

	f1 := frame.New(frame.MESSAGE,
		frame.Destination, "/queue/1")

	f2 := frame.New(frame.MESSAGE,
		frame.Destination, "/queue/2")

	f3 := frame.New(frame.MESSAGE,
		frame.Destination, "/queue/3")

	f4 := frame.New(frame.MESSAGE,
		frame.Destination, "/queue/4")

	err = txs.Add("tx1", f1)
	require.NoError(t, err)
	err = txs.Add("tx1", f2)
	require.NoError(t, err)
	err = txs.Add("tx1", f3)
	require.NoError(t, err)
	err = txs.Add("tx2", f4)

	var tx1 []*frame.Frame

	txs.Commit("tx1", func(f *frame.Frame) error {
		tx1 = append(tx1, f)
		return nil
	})
	require.NoError(t, err)

	var tx2 []*frame.Frame

	err = txs.Commit("tx2", func(f *frame.Frame) error {
		tx2 = append(tx2, f)
		return nil
	})
	require.NoError(t, err)

	require.Len(t, tx1, 3)
	require.Equal(t, f1, tx1[0])
	require.Equal(t, f2, tx1[1])
	require.Equal(t, f3, tx1[2])

	require.Len(t, tx2, 1)
	require.Equal(t, f4, tx2[0])

	// already committed, so should cause an error
	err = txs.Commit("tx1", func(f *frame.Frame) error {
		t.Fatal("should not be called")
		return nil
	})
	require.ErrorIs(t, err, txUnknown)
}
