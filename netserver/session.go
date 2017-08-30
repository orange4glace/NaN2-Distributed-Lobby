package netserver

import (
	"strings"
	"sync"
	"time"
)

type Session struct {
	token  string
	userid string
	time   time.Time
	conn   *Connection
	mutex  sync.Mutex
}

func newSession(userid string) *Session {
	s := new(Session)
	s.userid = userid
	s.time = time.Unix(0, 0)
	return s
}

func (sess *Session) Token() string {
	return sess.token
}

func (sess *Session) Lock() {
	sess.mutex.Lock()
}

func (sess *Session) Unlock() {
	sess.mutex.Unlock()
}

func (sess *Session) setToken(token string, time time.Time) bool {
	sess.mutex.Lock()
	defer sess.mutex.Unlock()
	if time.Before(sess.time) {
		return false
	}
	sess.token = token
	sess.time = time
	if sess.conn != nil {
		sess.conn.Write([]byte("session changed"))
		sess.conn.close()
		sess.conn = nil
	}
	return true
}

func (sess *Session) bindConnection(token string, conn *Connection) bool {
	sess.mutex.Lock()
	defer sess.mutex.Unlock()
	if strings.Compare(token, sess.token) != 0 {
		return false
	}
	sess.conn = conn
	return true
}
