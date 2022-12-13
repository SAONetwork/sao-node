package cache

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type Data struct {
	name   string
	length uint
}

func TestLruCache(t *testing.T) {
	svc := NewLruCacheSvc()

	svc.CreateCache("test1", 3)
	svc.CreateCache("test2", 2)

	svc.Put("test1", "aaa", &Data{name: "aaa", length: 100})
	svc.Put("test1", "bbb", &Data{name: "bbb", length: 200})
	svc.Put("test1", "ccc", &Data{name: "ccc", length: 300})

	data, err2 := svc.Get("test1", "aaa")
	require.NoError(t, err2)
	t.Logf("Data: %v", data)
	require.Equal(t, "aaa", data.(*Data).name)

	data, err2 = svc.Get("test1", "ccc")
	require.NoError(t, err2)
	require.Equal(t, "ccc", data.(*Data).name)

	svc.Put("test1", "ddd", &Data{name: "ddd", length: 400})

	data, err2 = svc.Get("test1", "ddd")
	require.NoError(t, err2)
	require.Equal(t, "ddd", data.(*Data).name)

	size := svc.GetSize("test1")
	require.Equal(t, 3, size)
	t.Logf("Data: %v", size)

	data, err2 = svc.Get("test1", "bbb")
	require.NoError(t, err2)
	require.Nil(t, data)

	svc.Put("test2", "eee", &Data{name: "eee", length: 200})
	svc.Put("test2", "fff", &Data{name: "fff", length: 300})
	svc.Put("test2", "ggg", &Data{name: "ggg", length: 300})
	svc.Put("test2", "hhh", &Data{name: "hhh", length: 300})

	size = svc.GetCapacity("test2")
	require.Equal(t, 2, size)
	t.Logf("Data: %v", size)

	svc.Evict("test2", "hhh")

	size = svc.GetSize("test2")
	require.Equal(t, 1, size)
	t.Logf("Data: %v", size)

	data, err2 = svc.Get("test2", "eee")
	require.NoError(t, err2)
	require.Nil(t, data)

	data, err2 = svc.Get("test2", "ggg")
	require.NoError(t, err2)
	require.Equal(t, "ggg", data.(*Data).name)
}
