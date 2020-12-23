package common

import (
	"encoding/json"
	"github.com/lucas-clemente/quic-go"
)

const Addr = "0.0.0.0:4242"

type CmdType int
const (
	ILLEGAL CmdType = iota
	REGISTER
	HTTP
)

var commands = []string{
	ILLEGAL:  "Illegal",
	REGISTER: "Register",
	HTTP:     "Http",
}

func (cmd CmdType) String() string {
	if cmd >= 0 && cmd < CmdType(len(commands)) {
		return commands[cmd]
	}
	return "Not defined command type."
}

type ResponseCode int
const (
	OK ResponseCode = iota
	BAD_REQUEST
	ERROR_FOUND
)

var responseDesc = []string{
	OK:          "OK",
	BAD_REQUEST: "Bad request content",
	ERROR_FOUND: "Internal error when processing request",
}

//const IDENTIFIER string = "identifier"

func (rc ResponseCode) String() string {
	if rc >= 0 && rc < ResponseCode(len(responseDesc)) {
		return responseDesc[rc]
	}
	return "Not defined response code."
}

type Command struct {
	Identifier string
	CType      CmdType
	Payload    interface{}
}

func (c *Command) Json() []byte {
	j, _ := json.Marshal(c)
	return j
}

type Response struct {
	Identifier string
	Code       ResponseCode
	Contents   interface{}
}

func (r *Response) Json() []byte {
	j, _ := json.Marshal(r)
	return j
}

type QuicConnection struct {
	Session quic.Session
	Stream  quic.Stream
}

func (qc *QuicConnection) IssueCommand(cmd Command) error{
	j := cmd.Json()
	if _, err := NewWriter(qc.Stream).Write(j); err != nil {
		return err
	} else {
		Log.Infof("The command %s is issued successfully", j)
	}
	return nil
}

type QConnectionManager map[string]*QuicConnection

var qm QConnectionManager = make(map[string]*QuicConnection)

func GetManager() QConnectionManager {
	return qm
}

func (qcm QConnectionManager) AddConn(id string, qc *QuicConnection) {
	qcm[id] = qc
}

func (qcm QConnectionManager) RemoveConn(id string) {
	delete(qcm, id)
}

func (qcm QConnectionManager) GetConn(id string) *QuicConnection {
	return qcm[id]
}

type EventType int

const (
	UNKNOWN EventType = iota
	RESPONSE
	COMMAND
	ERROR
)


type Listener interface {
	OnEvent(t EventType, payload []byte) error
}

type Subject interface {
	Add(Listener Listener)
	Remove(Listener Listener)
	NotifyAll(t EventType, payload []byte)
}

type ListenerMgr struct {
	listeners []Listener
}

func (lm *ListenerMgr) Add(l Listener) {
	lm.listeners = append(lm.listeners, l)
}

func (lm *ListenerMgr) Remove(l Listener) {
	index := -1
	for i, v := range lm.listeners {
		if v == l {
			index = i
			break
		}
	}
	if index != -1 {
		lm.listeners = remove(lm.listeners, index)
	}
}

func (lm *ListenerMgr) NotifyAll(t EventType, payload []byte) {
	for _, l := range lm.listeners {
		go func() {
			if err := l.OnEvent(t, payload); err != nil {
				Log.Errorf("%s", err)
			}
		}()
	}
}

func remove(listeners []Listener, s int) []Listener {
	return append(listeners[:s], listeners[s+1:]...)
}