package main

import (
	"fmt"
	"github.com/heyujiang/hrpc/client"
	"github.com/heyujiang/hrpc/server"
	"log"
	"net"
	"sync"
)

func startServer(addr chan string) {
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
			args := fmt.Sprintf("hrpc req %d", i)
			var reply string
			if err := c.Call("Foo.Sum", args, &reply); err != nil {
				log.Fatal("call Foo.Sum error: ", err)
			}
			log.Println("reply: ", reply)
		}(i)
	}

	wg.Wait()

}
