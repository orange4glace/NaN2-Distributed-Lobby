package sessmod

import (
	"log"

	"github.com/orange4glace/distributed-lobby/netserver"
)

type MySessionModule struct {
	netserver.SessionModule
}

func (sessmod *MySessionModule) OnMessageReceived(conn *netserver.Connection) {
	log.Print("Hello messsage!\n")
}
