package gateway

type RequestQueue []*WorkRequest

func (q RequestQueue) Len() int { return len(q) }

func (q *RequestQueue) Push(x *WorkRequest) {
	item := x
	*q = append(*q, item)
}

func (q *RequestQueue) Remove(i int) *WorkRequest {
	old := *q
	n := len(old)
	item := old[i]
	old[i] = old[n-1]
	old[n-1] = nil
	*q = old[0 : n-1]
	return item
}
