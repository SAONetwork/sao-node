package queue

import (
	"sync"

	"github.com/SaoNetwork/sao-node/types"
)

type WorkRequest struct {
	Order types.OrderInfo
	Shard types.ShardInfo
	Job   *types.Job
}

type RequestQueue struct {
	sync.Mutex
	queue []*WorkRequest
}

func (q *RequestQueue) Len() int {
	q.Lock()
	defer q.Unlock()

	return len(q.queue)
}

func (q *RequestQueue) Push(x *WorkRequest) {
	q.Lock()
	defer q.Unlock()

	item := x
	q.queue = append(q.queue, item)
}

func (q *RequestQueue) PopFront() *WorkRequest {
	q.Lock()
	defer q.Unlock()
	if len(q.queue) == 0 {
		return nil
	}

	item := q.queue[0]
	q.queue = q.queue[1:]
	return item
}

func (q *RequestQueue) Clean() {
	q.Lock()
	defer q.Unlock()

	q.queue = []*WorkRequest{}
}
