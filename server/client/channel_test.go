package client

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChannelWhenClosed(t *testing.T) {

	ch := make(chan int, 10)

	ch <- 1
	ch <- 2

	select {
	case i, ok := <-ch:
		require.Equal(t, 1, i)
		require.True(t, ok)
	default:
		t.Fatal("expected value on channel")
	}

	select {
	case i := <-ch:
		require.Equal(t, 2, i)
	default:
		t.Fatal("expected value on channel")
	}

	select {
	case _ = <-ch:
		t.Fatal("not expecting anything on the channel")
	default:
	}

	ch <- 3
	close(ch)

	select {
	case i := <-ch:
		require.Equal(t, 3, i)
	default:
		t.Fatal("expected value on channel")
	}

	select {
	case _, ok := <-ch:
		require.False(t, ok)
	default:
		t.Fatal("expected value on channel")
	}

	select {
	case _, ok := <-ch:
		require.False(t, ok)
	default:
		t.Fatal("expected value on channel")
	}
}

func TestChannelMultipleChannels(t *testing.T) {

	ch1 := make(chan int, 10)
	ch2 := make(chan string, 10)

	ch1 <- 1

	select {
	case i, ok := <-ch1:
		require.Equal(t, 1, i)
		require.True(t, ok)
	case _ = <-ch2:
	default:
		t.Fatal("expected value on channel")
	}

	select {
	case _ = <-ch1:
		t.Fatal("not expected")
	case _ = <-ch2:
		t.Fatal("not expected")
	default:
	}
}
