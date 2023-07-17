package cache

import (
	"fmt"
	"testing"
)

func TestLRU(t *testing.T) {
	c := CreateLruCache(2)
	c.put("a", 1)
	c.put("b", 1)
	c.put("c", 1)
	c.put("d", 1)

	c.get("a")
	c.get("b")
	c.get("c")
	c.get("d")

	fmt.Println(c.Size)
}
