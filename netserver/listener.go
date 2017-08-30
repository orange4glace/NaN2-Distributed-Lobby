package netserver

import (
	"log"
	"net"
	"strconv"
)

// Connector struct
type Listener struct {
	listener *net.TCPListener
	modules  []SessionModule
}

// New Connector instance
func NewListener() *Listener {
	lis := new(Listener)
	return lis
}

// Listen TCP
func (this *Listener) Listen(port int) {
	this.ListenNet(port, "tcp")
}

// ListenNet net-type
func (this *Listener) ListenNet(port int, nettype string) (err error) {
	addr, err := net.ResolveTCPAddr(nettype, "127.0.0.1"+":"+strconv.Itoa(port))
	if err != nil {
		return err
	}
	this.listener, err = net.ListenTCP(nettype, addr)
	if err != nil {
		return err
	}
	log.Printf("Listen port %d..\n", port)
	return
}

func (this *Listener) Accept() (*Connection, error) {
	conn, err := this.listener.AcceptTCP()
	if err != nil {
		return nil, err
	}
	wrappedConn := NewConnection(this, conn)
	return wrappedConn, nil
}

func (this *Listener) AttachSessionModule(sessmod SessionModule) {
	this.modules = append(this.modules, sessmod)
}

func (this *Listener) GetSessionModules() []SessionModule {
	return this.modules
}
