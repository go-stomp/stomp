package client

import (
	"container/list"

	"github.com/ingelectus/stomp/frame"
)

type txStore struct {
	transactions map[string]*list.List
}

// Initializes a new store or clears out an existing store
func (txs *txStore) Init() {
	txs.transactions = nil
}

func (txs *txStore) Begin(tx string) error {
	if txs.transactions == nil {
		txs.transactions = make(map[string]*list.List)
	}

	if _, ok := txs.transactions[tx]; ok {
		return txAlreadyInProgress
	}

	txs.transactions[tx] = list.New()
	return nil
}

func (txs *txStore) Abort(tx string) error {
	if list, ok := txs.transactions[tx]; ok {
		list.Init()
		delete(txs.transactions, tx)
		return nil
	}
	return txUnknown
}

// Commit causes all requests that have been queued for the transaction
// to be sent to the request channel for processing. Calls the commit
// function (commitFunc) in order for each request that is part of the
// transaction.
func (txs *txStore) Commit(tx string, commitFunc func(f *frame.Frame) error) error {
	if list, ok := txs.transactions[tx]; ok {
		for element := list.Front(); element != nil; element = list.Front() {
			err := commitFunc(list.Remove(element).(*frame.Frame))
			if err != nil {
				return err
			}
		}
		delete(txs.transactions, tx)
		return nil
	}
	return txUnknown
}

func (txs *txStore) Add(tx string, f *frame.Frame) error {
	if list, ok := txs.transactions[tx]; ok {
		f.Header.Del(frame.Transaction)
		list.PushBack(f)
		return nil
	}
	return txUnknown
}
