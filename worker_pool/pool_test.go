package worker_pool

import (
	"context"
	"errors"
	"fmt"
	"log"
	"testing"
)

func TestPool(t *testing.T) {
	run()
}

func run() {
	pool := NewPool(context.Background(), Conf{}, &TestDeal{})

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
	}
}

type TestDeal struct {
}

func (t *TestDeal) handler(data interface{}, index uint) error {
	fmt.Println(index, ": ", data.(int))
	if data.(int)%30 == 0 {
		return errors.New("test error")
	}
	return nil
}

func (t *TestDeal) deadHandler(data interface{}) error {
	fmt.Println(" dead : ", data.(int))
	return nil
}
