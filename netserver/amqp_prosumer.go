package netserver

import (
	"sync"

	"github.com/orange4glace/prosumer"
)

type AMQPProsumer struct {
	prosumer       *prosumer.Prosumer
	unacks         map[int]interface{}
	confirmCounter int
	mutex          sync.Mutex
}

func NewAMQPProsumer() *AMQPProsumer {
	ap := new(AMQPProsumer)
	ap.prosumer = prosumer.NewProsumer()
	ap.unacks = make(map[int]interface{})
	return ap
}

func (ap *AMQPProsumer) Produce(d interface{}) bool {
	ap.mutex.Lock()
	defer ap.mutex.Unlock()
	return ap.prosumer.Producer.Produce(d)
}

func (ap *AMQPProsumer) Consume() <-chan interface{} {
	r := make(chan interface{})
	ch := ap.prosumer.Consumer.Consume()
	go func() {
		for {
			val := <-ch
			ap.mutex.Lock()
			ap.confirmCounter++
			ap.unacks[ap.confirmCounter] = val
			ap.mutex.Unlock()
			r <- val
		}
	}()
	return r
}

func (ap *AMQPProsumer) Ack(seq int) bool {
	ap.mutex.Lock()
	defer ap.mutex.Unlock()
	if _, ok := ap.unacks[seq]; ok {
		delete(ap.unacks, seq)
		return true
	}
	return false
}

func (ap *AMQPProsumer) Reconfirm() {
	ap.mutex.Lock()
	defer ap.mutex.Unlock()
	for _, v := range ap.unacks {
		ap.prosumer.Producer.Produce(v)
	}
	ap.confirmCounter = 0
	ap.unacks = make(map[int]interface{})
}
