package cache

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type Data struct {
	name   string
	length uint
}

func TestCache(t *testing.T) {
	svc := NewCacheSvc()

	svc.CreateCache("test1", 3)
	svc.CreateCache("test2", 2)

	err := svc.Put("test1", "aaa", &Data{name: "aaa", length: 100})
	require.NoError(t, err)

	err = svc.Put("test1", "bbb", &Data{name: "bbb", length: 200})
	require.NoError(t, err)

	err = svc.Put("test1", "ccc", &Data{name: "ccc", length: 300})
	require.NoError(t, err)

	data, err2 := svc.Get("test1", "aaa")
	require.NoError(t, err2)
	t.Logf("Data: %v", data)
	require.Equal(t, "aaa", data.(*Data).name)

	data, err2 = svc.Get("test1", "ccc")
	require.NoError(t, err2)
	require.Equal(t, "ccc", data.(*Data).name)

	err2 = svc.Put("test1", "ddd", &Data{name: "ddd", length: 400})
	require.NoError(t, err2)

	data, err2 = svc.Get("test1", "ddd")
	require.NoError(t, err2)
	require.Equal(t, "ddd", data.(*Data).name)

	size, err3 := svc.GetSize("test1")
	require.NoError(t, err3)
	require.Equal(t, 3, size)
	t.Logf("Data: %v", size)

	data, err2 = svc.Get("test1", "bbb")
	require.NoError(t, err2)
	require.Nil(t, data)

	err = svc.Put("test2", "eee", &Data{name: "eee", length: 200})
	require.NoError(t, err)

	err = svc.Put("test2", "fff", &Data{name: "fff", length: 300})
	require.NoError(t, err)

	err = svc.Put("test2", "ggg", &Data{name: "ggg", length: 300})
	require.NoError(t, err)

	err = svc.Put("test2", "hhh", &Data{name: "hhh", length: 300})
	require.NoError(t, err)

	size, err3 = svc.GetCapacity("test2")
	require.NoError(t, err3)
	require.Equal(t, 2, size)
	t.Logf("Data: %v", size)

	err = svc.Evict("test2", "hhh")
	require.NoError(t, err)

	size, err3 = svc.GetSize("test2")
	require.NoError(t, err3)
	require.Equal(t, 1, size)
	t.Logf("Data: %v", size)

	data, err2 = svc.Get("test2", "eee")
	require.NoError(t, err2)
	require.Nil(t, data)

	data, err2 = svc.Get("test2", "ggg")
	require.NoError(t, err2)
	require.Equal(t, "ggg", data.(*Data).name)
}
