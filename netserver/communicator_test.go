package netserver

import (
	"log"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/streadway/amqp"
)

func TestCommunicator(t *testing.T) {
	lis := NewListener()
	lis.Listen(6666)
	go func() {
		for {
			println("Waiting for connection..")
			conn, err := lis.Accept()
			if err != nil {
				continue
			}
			go func() {
				conn.Read()
			}()
		}
	}()

	c := newCommunicator()
	c.Open()
	go func() {
		time.Sleep(1 * time.Second)
		conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
		if err != nil {
			return
		}
		ch, _ := conn.Channel()
		q, _ := ch.QueueDeclare("", false, false, true, false, nil)
		go func() {
			con, _ := ch.Consume(q.Name, "", true, true, false, false, nil)
			for d := range con {
				log.Printf("Callback %s", d.Body)
			}
		}()
		uuid, _ := uuid.NewUUID()
		corr := uuid.String()
		ch.Publish("", "session_queue", false, false, amqp.Publishing{
			ReplyTo:       q.Name,
			CorrelationId: corr,
			Body:          []byte("abc def"),
		})
	}()
	time.Sleep(5000 * time.Second)
}

func TestExchangeDeclare(t *testing.T) {
	conn, _ := amqp.Dial("amqp://guest:guest@localhost:5672/")
	ch1, _ := conn.Channel()
	ch2, _ := conn.Channel()
	ch1.Confirm(false)
	ch2.Confirm(false)
	err := ch1.ExchangeDeclare("abcd", "fanout", false, false, false, false, nil)
	if err != nil {
		log.Printf("%s", err.Error())
	}
	go func() {
		for {
			log.Printf("Publishing..")
			ch1.Publish("abcd", "", false, false, amqp.Publishing{
				DeliveryMode: 2,
				ContentType:  "text/plain",
				Body:         []byte("hello"),
			})
			time.Sleep(1 * time.Second)
		}
	}()
	err = ch2.ExchangeDeclare("abcd", "fanout", false, false, false, false, nil)
	if err != nil {
		log.Printf("%s", err.Error())
	}
	q, _ := ch2.QueueDeclare("", false, false, true, false, nil)
	ch2.QueueBind(q.Name, "", "abcd", false, nil)
	for {
		log.Printf("Consuming..")
		g, _ := ch2.Consume(q.Name, "", false, false, false, false, nil)
		for {
			<-g
			log.Printf("AAAAAAAAAAAAAAA")
		}
	}
}
