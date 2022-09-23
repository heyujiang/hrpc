package worker_pool

import (
	"context"
	"errors"
	"log"
	"sync"
)

var ErrPoolClosed = errors.New("worker pool is closed")

type Deal interface {
	deal(data interface{}, index uint)
}

type Pool struct {
	task      chan interface{}
	workerNum uint
	mu        sync.Mutex
	closed    bool
	dealF     Deal
	ctx       context.Context
	done      context.CancelFunc
}

func NewPool(ctx context.Context, num uint, deal Deal) *Pool {
	c, cancelF := context.WithCancel(ctx)
	return &Pool{
		task:      make(chan interface{}),
		workerNum: num,
		mu:        sync.Mutex{},
		closed:    false,
		dealF:     deal,
		ctx:       c,
		done:      cancelF,
	}
}

func (p *Pool) Start() error {
	if p.closed {
		return ErrPoolClosed
	}
	var i uint
	for ; i < p.workerNum; i++ {
		go p.deal(p.ctx, i)
	}
	return nil
}

func (p *Pool) Stop() error {
	if p.closed {
		return ErrPoolClosed
	}

	p.done()
	p.closed = true
	return nil
}

func (p *Pool) deal(ctx context.Context, index uint) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker pool index:%d is closed !!!\r\n", index)
			return
		case d := <-p.task:
			p.dealF.deal(d, index)
		default:
		}
	}
}

func (p *Pool) Distribute(data interface{}) error {
	if p.closed {
		return ErrPoolClosed
	}
	p.task <- data
	return nil
}
