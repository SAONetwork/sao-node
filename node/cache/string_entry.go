package cache

import (
	hamt "github.com/raviqqe/hamt"
)

type entryString string

// FNV Hash prime
const primeRK = 16777619

func (s entryString) Hash() uint32 {
	return GetHash32(s)
}

func (i entryString) Equal(e hamt.Entry) bool {
	j, ok := e.(entryString)

	if !ok {
		return false
	}

	return i == j
}

func GetHash32(dataStr entryString) uint32 {
	data := []byte(dataStr)

	hash := uint32(0)
	for i := 0; i < len(data); i++ {
		//hash = 0 * 16777619 + sep[i]
		hash = hash*primeRK + uint32(data[i])
	}
	var pow, sq uint32 = 1, primeRK
	for i := len(data); i > 0; i >>= 1 {
		if i&1 != 0 {
			pow *= sq
		}
		sq *= sq
	}
	return hash
}
