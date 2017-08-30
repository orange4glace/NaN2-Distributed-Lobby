package netserver

import "time"

type sessionMessage struct {
	userid  string
	token   string
	time    time.Time
	replyTo string
	corrID  string
	retry   int
	ok      bool
}

func (sm *sessionMessage) string() string {
	return sm.userid + " " + sm.token + " " + sm.time.String()
}
