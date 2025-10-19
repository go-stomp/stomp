package frame

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHeaderGetSetAddDel(t *testing.T) {
	h := &Header{}
	require.Equal(t, "", h.Get("xxx"))
	h.Add("xxx", "yyy")
	require.Equal(t, "yyy", h.Get("xxx"))
	h.Add("xxx", "zzz")
	require.Equal(t, []string{"yyy", "zzz"}, h.GetAll("xxx"))
	h.Set("xxx", "111")
	require.Equal(t, h.Get("xxx"), "111")
	h.Del("xxx")
	require.Equal(t, "", h.Get("xxx"))
}

func TestHeaderClone(t *testing.T) {
	h := Header{}
	h.Set("xxx", "yyy")
	h.Set("yyy", "zzz")

	hc := h.Clone()
	h.Del("xxx")
	h.Del("yyy")
	require.Equal(t, "yyy", hc.Get("xxx"))
	require.Equal(t, "zzz", hc.Get("yyy"))
}

func TestHeaderContains(t *testing.T) {
	h := NewHeader("xxx", "yyy", "zzz", "aaa", "xxx", "ccc")
	v, ok := h.Contains("xxx")
	require.Equal(t, "yyy", v)
	require.True(t, ok)

	v, ok = h.Contains("123")
	require.Equal(t, v, "")
	require.False(t, ok)
}

func TestHeaderContentLength(t *testing.T) {
	h := NewHeader("xxx", "yy", "content-length", "202", "zz", "123")
	cl, ok, err := h.ContentLength()
	require.Equal(t, 202, cl)
	require.True(t, ok)
	require.NoError(t, err)

	h.Set("content-length", "twenty")
	cl, ok, err = h.ContentLength()
	require.Equal(t, 0, cl)
	require.True(t, ok)
	require.Error(t, err)

	h.Del("content-length")
	cl, ok, err = h.ContentLength()
	require.Equal(t, 0, cl)
	require.False(t, ok)
	require.NoError(t, err)
}

func TestHeaderLit(t *testing.T) {
	_ = Frame{
		Command: "CONNECT",
		Header:  NewHeader("login", "xxx", "passcode", "yyy"),
		Body:    []byte{1, 2, 3, 4},
	}
}
