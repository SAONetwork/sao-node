package chain

import "sync"

type SeqCounter struct {
	sync.Mutex
	seq uint64
}

func NewSeqCounter(seq uint64) SeqCounter {
	return SeqCounter{seq: seq}
}

func (s SeqCounter) GetSeq() uint64 {
	s.Lock()
	defer s.Unlock()

	ret := s.seq
	s.seq++
	return ret
}
