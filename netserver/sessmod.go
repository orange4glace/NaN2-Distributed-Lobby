package netserver

type SessionModule interface {
	OnMessageReceived(conn *Connection, buf []byte, len int) bool
}
