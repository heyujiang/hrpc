package worker_pool

import (
	"context"
	"fmt"
	"log"
	"testing"
)

func TestPool(t *testing.T) {
	run()
}

func run() {
	pool := NewPool(context.Background(), 10, &TestDeal{})

	_ = pool.Start()
	defer func() {
		_ = pool.Stop()
	}()
	for i := 0; i < 1000; i++ {
		err := pool.Distribute(i)
		if err != nil {
			log.Println(err.Error())
			return
		}
		//if i == 567 {
		//	_ = pool.Stop()
		//}
	}
	//_ = pool.Stop()
}

type TestDeal struct {
}

func (t *TestDeal) deal(data interface{}, index uint) {
	fmt.Println(index, ": ", data.(int))
}
