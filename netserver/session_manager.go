package netserver

import (
	"errors"
	"sync"
)

var sessionMapMutex sync.Mutex
var sessionMessageMutex sync.Mutex
var sessionMap map[string]*Session
var sessionMessageMap map[int]*sessionMessage

func init() {
	sessionMap = make(map[string]*Session)
	sessionMessageMap = make(map[int]*sessionMessage)
}

func lockSessions() {
	sessionMapMutex.Lock()
}

func unlockSessions() {
	sessionMapMutex.Unlock()
}

func setSession(userid string, token string) {
	lockSessions()
	defer unlockSessions()
	sess := sessionMap[userid]
	if sess == nil {
		sess = new(Session)
		sess.userid = userid
		sess.token = token
		sessionMap[userid] = sess
	} else {
		sess.token = token
	}
}

func getSession(userid string) *Session {
	lockSessions()
	defer unlockSessions()
	if val, ok := sessionMap[userid]; ok {
		return val
	}
	sess := newSession(userid)
	sessionMap[userid] = sess
	return sess
}

func deleteSession(userid string) bool {
	lockSessions()
	defer unlockSessions()
	sess := sessionMap[userid]
	if sess == nil {
		return false
	}
	sess.Lock()
	defer sess.Unlock()
	delete(sessionMap, userid)
	return true
}

func addSessionConfirm(seq int, sm *sessionMessage) error {
	sessionMessageMutex.Lock()
	defer sessionMessageMutex.Unlock()
	_, ok := sessionMessageMap[seq]
	if ok {
		return errors.New("Confirmation exist")
	}
	sessionMessageMap[seq] = sm
	return nil
}

func confirmSessionConfirm(seq int) error {
	sessionMessageMutex.Lock()
	defer sessionMessageMutex.Unlock()
	_, ok := sessionMessageMap[seq]
	if !ok {
		return errors.New("Confirmation does not exist")
	}
	delete(sessionMessageMap, seq)
	return nil
}
