package netserver

import (
	"log"
	"strings"
	"time"

	"sync"

	"github.com/google/uuid"
	"github.com/orange4glace/prosumer"
	"github.com/orange4glace/reliableamqp"
	"github.com/streadway/amqp"
)

type Communicator struct {
	amqpConn *reliableamqp.Connection

	broadcasteeCloser   sync.WaitGroup
	broadcasteeProsumer *prosumer.Prosumer

	broadcasterCloser   sync.WaitGroup
	broadcasterProsumer *AMQPProsumer

	receiverCloser   sync.WaitGroup
	receiverProsumer *AMQPProsumer

	senderCloser   sync.WaitGroup
	senderProsumer *prosumer.Prosumer
}

func newCommunicator() (c *Communicator) {
	c = new(Communicator)
	c.amqpConn = reliableamqp.NewConnection()
	c.broadcasterProsumer = NewAMQPProsumer()
	c.broadcasteeProsumer = prosumer.NewProsumer()
	c.senderProsumer = prosumer.NewProsumer()
	return c
}

func (c *Communicator) Open() {
	c.amqpConn.OnOpened = c.onAMQPConnectionOpened
	c.amqpConn.OnClosed = c.onAMQPConnectionClosed

	c.amqpConn.Open("amqp://guest:guest@localhost:5672/", 1000)

	c.amqpConn.ReliableChannel(c.onBroadcasterChannelOpened, c.onBroadcasterChannelClosed)
	c.amqpConn.ReliableChannel(c.onBroadcasteeChannelOpened, c.onBroadcasteeChannelClosed)
	c.amqpConn.ReliableChannel(c.onReceiverChannelOpened, c.onReceiverChannelClosed)
	c.amqpConn.ReliableChannel(c.onSenderChannelOpened, c.onSenderChannelClosed)
}

func (c *Communicator) onAMQPConnectionOpened(amqp *reliableamqp.Connection) {
}

func (c *Communicator) onAMQPConnectionClosed(err *amqp.Error) {

}

func (c *Communicator) onBroadcasterChannelOpened(ch *reliableamqp.Channel, done chan<- bool) {
	log.Printf("# Broadcaster channel open")
	err := ch.Confirm(false)
	if err != nil {
		ch.Cl()
		done <- false
		return
	}
	err = ch.ExchangeDeclare("session_broadcast_exchange", "fanout", false, true, false, false, nil)
	if err != nil {
		log.Fatalf("%s", err.Error())
		ch.Cl()
		done <- false
		return
	}
	consumer := c.broadcasterProsumer.Consume()
	c.broadcasterCloser.Add(1)
	go func() {
		defer c.broadcasterCloser.Done()
		closeCh := ch.NotifyClose(make(chan *amqp.Error))
		closeChE := make(chan *amqp.Error)
		go func() {
			closeChE <- <-closeCh
		}()
		confirmSeq := 1
		confirmCh := ch.NotifyPublish(make(chan amqp.Confirmation, 1))
	outer:
		for {
			select {
			case item := <-consumer:
				sessMsg := item.(*sessionMessage)
				log.Printf("Publish %s", sessMsg.userid)
				err := ch.Publish("session_broadcast_exchange", "", false, false, amqp.Publishing{
					Body: []byte(sessMsg.string()),
				})
				if err != nil {
					log.Println("Error", err.Error())
					ch.Cl()
					return
				}
				log.Println("Waiting for ack..")
				confirmation := <-confirmCh
				if confirmation.Ack {
					log.Printf("Acked")
					sess := getSession(sessMsg.userid)
					sess.setToken(sessMsg.token, sessMsg.time)
					c.sendResponseToReceiver(sessMsg)
				} else {
					log.Printf("Nacked")
					go func() {
						time.Sleep(100 * time.Millisecond)
						sessMsg.retry++
						c.broadcasterProsumer.Produce(sessMsg)
					}()
				}
				confirmSeq++
			case err := <-closeChE:
				log.Println("Publish closed error", err.Error())
				break outer
			}
		}
	}()
	done <- true
}

func (c *Communicator) onBroadcasterChannelClosed(err *amqp.Error, done chan<- bool) {
	log.Printf("# Broadcaster channel close")
	c.broadcasteeCloser.Wait()
	done <- true
}

func (c *Communicator) onBroadcasteeChannelOpened(ch *reliableamqp.Channel, done chan<- bool) {
	log.Printf("# BroadcasteE channel open")
	err := ch.Confirm(false)
	if err != nil {
		ch.Cl()
		done <- false
		return
	}
	err = ch.ExchangeDeclare("session_broadcast_exchange", "fanout", false, true, false, false, nil)
	if err != nil {
		log.Fatalf("%s", err.Error())
		ch.Cl()
		done <- false
		return
	}
	q, err := ch.QueueDeclare(
		"",
		false,
		true,
		false,
		false,
		nil)
	if err != nil {
		log.Fatalf("%s", err.Error())
		ch.Cl()
		done <- false
		return
	}
	log.Println("# Broadcastee queue", q.Name)
	err = ch.QueueBind(q.Name, "", "session_broadcast_exchange", false, nil)
	if err != nil {
		log.Fatalf("%s", err.Error())
		ch.Cl()
		done <- false
		return
	}
	c.broadcasteeCloser.Add(1)
	go func() {
		defer c.broadcasteeCloser.Done()
		closeCh := ch.NotifyClose(make(chan *amqp.Error))
		consume, err := ch.Consume(q.Name, "", false, true, false, false, nil)
		if err != nil {
			ch.Cl()
			return
		}
	outer:
		for {
			log.Printf("Waiting for broadcastee..")
			select {
			case delivery := <-consume:
				body := string(delivery.Body)
				splits := strings.SplitN(body, " ", 2)
				userid := splits[0]
				tag := splits[1]
				log.Printf("Broadcastee : %s, %s", userid, tag)
			case <-closeCh:
				log.Printf("Broadcastee close")
				break outer
			}
		}
	}()
	done <- true
}

func (c *Communicator) onBroadcasteeChannelClosed(err *amqp.Error, done chan<- bool) {
	log.Printf("# BroadcasteE channel close")
	c.broadcasteeCloser.Wait()
	done <- true
}

func (c *Communicator) onReceiverChannelOpened(ch *reliableamqp.Channel, done chan<- bool) {
	log.Printf("# Receiver channel open")
	err := ch.Confirm(false)
	if err != nil {
		ch.Cl()
		done <- false
		return
	}
	q, err := ch.QueueDeclare(
		"session_queue",
		false,
		true,
		false,
		false,
		nil)
	if err != nil {
		log.Fatalf("%s", err.Error())
		ch.Cl()
		done <- false
		return
	}
	c.receiverCloser.Add(1)
	go func() {
		defer c.receiverCloser.Done()
		consume, err := ch.Consume(q.Name, "", false, false, false, false, nil)
		if err != nil {
			ch.Cl()
			return
		}
		for delivery := range consume {
			log.Printf("Waiting for receiver..")
			userid := string(delivery.Body)
			uuid, err := uuid.NewUUID()
			replyTo := delivery.ReplyTo
			if err != nil {
				continue
			}
			token := uuid.String()
			time := time.Now()
			log.Printf("Receiver : %s, %s %s", userid, replyTo, token)
			c.broadcastSession(userid, token, time, delivery.ReplyTo, delivery.CorrelationId)
		}
	}()
	done <- true
}

func (c *Communicator) onReceiverChannelClosed(err *amqp.Error, done chan<- bool) {
	log.Printf("# Receiver channel close")
	c.receiverCloser.Wait()
	done <- true
}

func (c *Communicator) onSenderChannelOpened(ch *reliableamqp.Channel, done chan<- bool) {
	consumer := c.senderProsumer.Consumer.Consume()
	c.senderCloser.Add(1)
	go func() {
		defer c.senderCloser.Done()
		closeCh := ch.NotifyClose(make(chan *amqp.Error))
	outer:
		for {
			select {
			case item := <-consumer:
				sessMsg := item.(*sessionMessage)
				log.Println("Send", sessMsg.userid)
				err := ch.Publish("", sessMsg.replyTo, false, false, amqp.Publishing{
					CorrelationId: sessMsg.corrID,
					Body:          []byte(sessMsg.string()),
				})
				if err != nil {
					c.senderProsumer.Producer.Produce(item)
					ch.Cl()
					return
				}
			case <-closeCh:
				break outer
			}
		}
	}()
	done <- true
}

func (c *Communicator) onSenderChannelClosed(err *amqp.Error, done chan<- bool) {
	c.senderCloser.Wait()
	done <- true
}

func (c *Communicator) broadcastSession(userid, token string, time time.Time, replyTo, corrID string) {
	c.broadcasterProsumer.Produce(&sessionMessage{
		userid:  userid,
		token:   token,
		time:    time,
		replyTo: replyTo,
		corrID:  corrID,
	})
}

func (c *Communicator) sendResponseToReceiver(sessMsg *sessionMessage) {
	c.senderProsumer.Producer.Produce(sessMsg)
}
