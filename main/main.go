package main

import (
	"github.com/heyujiang/hrpc/client"
	"github.com/heyujiang/hrpc/server"
	"log"
	"net"
	"sync"
)

type Foo int

type Args struct {
	Num1, Num2 int
}

func (f Foo) Sum(arg Args, reply *int) error {
	*reply = arg.Num1 + arg.Num2
	return nil
}

func startServer(addr chan string) {
	var foo Foo
	if err := server.Register(&foo); err != nil {
		log.Fatal("register error:", err)
	}

	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("network error: ", err)
	}
	log.Println("start rpc server on", l.Addr())
	addr <- l.Addr().String()
	server.Accept(l)
}

func main() {
	log.SetFlags(0)
	addr := make(chan string)
	go startServer(addr)

	c, _ := client.Dial("tcp", <-addr)
	defer func() {
		_ = c.Close()
	}()

	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := &Args{Num1: i, Num2: i * 1}
			var reply int
			if err := c.Call("Foo.Sum", args, &reply); err != nil {
				log.Fatal("call Foo.Sum error: ", err)
			}
			log.Printf("%d + %d = %d ", args.Num1, args.Num2, reply)
		}(i)
	}

	wg.Wait()

}
