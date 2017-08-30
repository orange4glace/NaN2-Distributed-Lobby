package main

import (
	"fmt"
	"testing"
	"time"
)

func TestChannel(t *testing.T) {
	ch := make(chan int)
	go func() {
		for m := range ch {
			fmt.Printf("%d", m)
		}
		fmt.Println("closed")
	}()
	go func() {
		for i := 5; i < 10; i++ {
			ch <- i
		}
		close(ch)
	}()
	time.Sleep(1 * time.Second)
}
