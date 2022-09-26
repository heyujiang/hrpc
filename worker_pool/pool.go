package worker_pool

import (
	"context"
	"errors"
	"log"
	"sync"
)

var (
	defaultWorkerNum uint = 5
	defaultMaxRetry  uint = 3
	ErrPoolClosed         = errors.New("worker pool is closed")
)

type Handler interface {
	handler(data interface{}, index uint) error
	deadHandler(data interface{}) error
}

type Pool struct {
	task      chan interface{}
	deadTask  chan interface{}
	workerNum uint
	maxRetry  uint
	mu        sync.Mutex
	closed    bool
	handler   Handler
	ctx       context.Context
	done      context.CancelFunc
}

type Conf struct {
	workerNum uint
	maxRetry  uint
}

func NewPool(ctx context.Context, conf Conf, handler Handler) *Pool {
	if conf.workerNum == 0 {
		conf.workerNum = defaultWorkerNum
	}
	if conf.maxRetry == 0 {
		conf.maxRetry = defaultMaxRetry
	}
	c, cancelF := context.WithCancel(ctx)

	return &Pool{
		task:      make(chan interface{}),
		deadTask:  make(chan interface{}),
		workerNum: conf.workerNum,
		maxRetry:  conf.maxRetry,
		mu:        sync.Mutex{},
		closed:    false,
		handler:   handler,
		ctx:       c,
		done:      cancelF,
	}
}

// Start 启动工作池，启动workerNum数量个worker协程
func (p *Pool) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return ErrPoolClosed
	}

	go p.doDead(p.ctx)

	var i uint
	for ; i < p.workerNum; i++ {
		go p.do(p.ctx, i)
	}
	return nil
}

// Stop 停止工作池
func (p *Pool) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return ErrPoolClosed
	}

	p.done()
	p.closed = true
	return nil
}

// Distribute 发布任务
func (p *Pool) Distribute(data interface{}) error {
	if p.closed {
		return ErrPoolClosed
	}
	p.task <- data
	return nil
}

func (p *Pool) sendDead(data interface{}) error {
	if p.closed {
		return ErrPoolClosed
	}
	p.deadTask <- data
	return nil
}

// worker调用业务方的业务执行方法
func (p *Pool) do(ctx context.Context, index uint) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker pool index:%d is closed !!!\r\n", index)
			return
		case d := <-p.task:
			var i uint
			for ; i < p.maxRetry; i++ {
				err := p.handler.handler(d, index)
				if err == nil {
					break
				} else {
					log.Printf("Task handler data:%v with %d error ：%s \r\n", d, i, err.Error())
				}
			}
			if i == p.maxRetry {
				err := p.sendDead(d)
				if err != nil {
					if err == ErrPoolClosed {
						return
					}
				}
			}
		}
	}
}

// worker调用业务方的业务执行方法
func (p *Pool) doDead(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker pool dead task: is closed !!!\r\n")
			return
		case d := <-p.deadTask:
			err := p.handler.deadHandler(d)
			if err != nil {
				log.Printf("Dead task handler error ：%s \r\n", err.Error())
			}
		}
	}
}
