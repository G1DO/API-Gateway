package lb

import "sync/atomic"

type Balancer interface {
	Next() string
}

type RoundRobin struct {
	backends []string
	counter  uint64
}

func NewRoundRobin(backends []string) *RoundRobin {
	return &RoundRobin{
		backends: backends,
	}
}

func (rr *RoundRobin) Next() string {
	idx := atomic.AddUint64(&rr.counter, 1)
	return rr.backends[idx%uint64(len(rr.backends))]
}