package client

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSubscriptionListAddAndGet(t *testing.T) {
	sub1 := newSubscription(nil, "/dest", "1", "client")
	sub2 := newSubscription(nil, "/dest", "2", "client")
	sub3 := newSubscription(nil, "/dest", "3", "client")

	sl := NewSubscriptionList()
	sl.Add(sub1)
	sl.Add(sub2)
	sl.Add(sub3)

	require.Equal(t, sub1, sl.Get())

	// add the subscription again, should go to the back
	sl.Add(sub1)

	require.Equal(t, sub2, sl.Get())
	require.Equal(t, sub3, sl.Get())
	require.Equal(t, sub1, sl.Get())

	require.Nil(t, sl.Get())
}

func TestSubscriptionListAddAndRemove(t *testing.T) {
	sub1 := newSubscription(nil, "/dest", "1", "client")
	sub2 := newSubscription(nil, "/dest", "2", "client")
	sub3 := newSubscription(nil, "/dest", "3", "client")

	sl := NewSubscriptionList()
	sl.Add(sub1)
	sl.Add(sub2)
	sl.Add(sub3)

	require.Equal(t, 3, sl.subs.Len())

	// now remove the second subscription
	sl.Remove(sub2)

	require.Equal(t, sub1, sl.Get())
	require.Equal(t, sub3, sl.Get())
	require.Nil(t, sl.Get())
}

func TestSubscriptionListAck(t *testing.T) {
	sub1 := &Subscription{dest: "/dest1", id: "1", ack: "client", msgId: 101}
	sub2 := &Subscription{dest: "/dest3", id: "2", ack: "client-individual", msgId: 102}
	sub3 := &Subscription{dest: "/dest4", id: "3", ack: "client", msgId: 103}
	sub4 := &Subscription{dest: "/dest4", id: "4", ack: "client", msgId: 104}

	sl := NewSubscriptionList()
	sl.Add(sub1)
	sl.Add(sub2)
	sl.Add(sub3)
	sl.Add(sub4)

	require.Equal(t, 4, sl.subs.Len())

	var subs []*Subscription
	callback := func(s *Subscription) {
		subs = append(subs, s)
	}

	// now remove the second subscription
	sl.Ack(103, callback)

	require.Len(t, subs, 2)
	require.Equal(t, sub1, subs[0])
	require.Equal(t, sub3, subs[1])

	require.Equal(t, sub2, sl.Get())
	require.Equal(t, sub4, sl.Get())
	require.Nil(t, sl.Get())
}

func TestSubscriptionListNack(t *testing.T) {
	sub1 := &Subscription{dest: "/dest1", id: "1", ack: "client", msgId: 101}
	sub2 := &Subscription{dest: "/dest3", id: "2", ack: "client-individual", msgId: 102}
	sub3 := &Subscription{dest: "/dest4", id: "3", ack: "client", msgId: 103}
	sub4 := &Subscription{dest: "/dest4", id: "4", ack: "client", msgId: 104}

	sl := NewSubscriptionList()
	sl.Add(sub1)
	sl.Add(sub2)
	sl.Add(sub3)
	sl.Add(sub4)

	require.Equal(t, 4, sl.subs.Len())

	var subs []*Subscription
	callback := func(s *Subscription) {
		subs = append(subs, s)
	}

	// now remove the second subscription
	sl.Nack(103, callback)

	require.Len(t, subs, 1)
	require.Equal(t, sub3, subs[0])

	require.Equal(t, sub1, sl.Get())
	require.Equal(t, sub2, sl.Get())
	require.Equal(t, sub4, sl.Get())
	require.Nil(t, sl.Get())
}
