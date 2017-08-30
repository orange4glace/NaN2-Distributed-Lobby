package netserver

import (
	"io"
	"log"
	"net"
	"strings"
	"time"
)

type Connection struct {
	listener      *Listener
	conn          *net.TCPConn
	authenticated bool
}

func NewConnection(listener *Listener, conn *net.TCPConn) *Connection {
	connection := new(Connection)
	connection.listener = listener
	connection.conn = conn

	return connection
}

func (conn *Connection) close() error {
	return conn.conn.Close()
}

func (conn *Connection) Write(body []byte) (int, error) {
	return conn.conn.Write(body)
}

func (conn *Connection) Read() {
	for {
		readTimeout := time.Now().Add(10 * time.Second)
		err := conn.conn.SetReadDeadline(readTimeout)
		if err != nil {
			log.Printf("Failed to SetReadDeadline. %s", err.Error())
			conn.close()
			return
		}
		log.Printf("Set ReadTimeout. %s", readTimeout.String())
		buf := make([]byte, 100)
		len, err := conn.conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				// Connection closed
				conn.close()
				return
			}
			log.Printf("Error while read buffer (%s)\n", err.Error())
			if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
				conn.close()
				return
			}
		}
		if conn.authenticated == false {
			res := conn.authenticate(buf, len)
			if res == false {
				conn.Write([]byte("401 unauthorized"))
				conn.close()
				return
			}
			conn.Write([]byte("Authenciated"))
		}
		mods := conn.listener.GetSessionModules()
		for _, mod := range mods {
			mod.OnMessageReceived(conn, buf, len)
		}
	}
}

func (conn *Connection) authenticate(buf []byte, len int) bool {
	if conn.authenticated == true {
		return true
	}
	str := string(buf[:len])
	splits := strings.SplitN(str, " ", 2)
	userid := splits[0]
	token := splits[1]
	sess := getSession(userid)
	res := sess.bindConnection(token, conn)
	if res {
		conn.authenticated = true
	}
	log.Println("Authenticate", userid, "t:", token, "st:", sess.token)
	return res
}
