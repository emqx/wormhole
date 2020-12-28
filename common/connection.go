package common

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/lucas-clemente/quic-go"
	"github.com/sirupsen/logrus"
	"net/http"
	"sync"
	"time"
)

const Addr = "0.0.0.0:4242"

type CmdType int
const (
	ILLEGAL CmdType = iota
	REGISTER
	HTTP
)

type ResponseCode int
const (
	OK ResponseCode = iota
	BAD_REQUEST
	ERROR_FOUND
)

type ResponseType int
const (
	BASIC_R  ResponseType = iota
	HTTP_R
)


type Command interface {
	GetSequence() int
	Json() []byte
	Validate() *BasicResponse
}

type BasicCommand struct {
	Identifier string
	Sequence   int
	CType      CmdType
}

type HttpCommand struct {
	BasicCommand
	HttpRequest
}

func (c *BasicCommand) GetSequence() int {
	return c.Sequence
}

func (c *BasicCommand) Json() []byte {
	j, _ := json.Marshal(c)
	return j
}

//Return a not empty response if the Identifier is empty
func (c *BasicCommand) Validate() *BasicResponse {
	return validateCmd(*c)
}

func validateCmd(cmd BasicCommand) *BasicResponse {
	if cmd.Identifier == "" {
		estr := fmt.Sprintf("Identifier is required for registration.")
		Log.Errorf("%s", estr)
		return &BasicResponse{
			Code:        BAD_REQUEST,
			Description: estr,
		}
	}
	return nil
}

func (c *HttpCommand) Json() []byte {
	j, _ := json.Marshal(c)
	return j
}

//Return a not empty response if the Identifier is empty
func (c *HttpCommand) Validate() *BasicResponse {
	return validateCmd(c.BasicCommand)
}

type Response interface {
	GetSequence() int
	GetResponseCode() ResponseCode
	GetDescription() string
	Json() []byte
	Validate() error
}

type BasicResponse struct {
	ResponseType ResponseType
	Identifier   string
	Sequence     int
	Code         ResponseCode
	Description  string
}

type HttpResponse struct {
	BasicResponse
	http.Header
	Body []byte
}

func (r *BasicResponse)GetSequence() int {
	return r.Sequence
}

func (r *BasicResponse)GetResponseCode() ResponseCode {
	return r.Code
}

func(r *BasicResponse)GetDescription() string {
	return r.Description
}

func (r *BasicResponse) Json() []byte {
	j, _ := json.Marshal(r)
	return j
}

func (r *BasicResponse)Validate() error {
	if r.Identifier == "" || r.Sequence == 0 {
		return fmt.Errorf("Invalid response, either identifier is empty or sequenct is 0.")
	}
	return nil
}

type QuicConnection struct {
	Session       quic.Session
	Stream        quic.Stream
	commandStatus map[int]*commandStatus
	Cancel        context.CancelFunc
}

type commandStatus struct {
	status   chan int
	response Response
	timeout  int64
	error    error
}

func (cs *commandStatus) start(wg *sync.WaitGroup) {
	for alive := true; alive; {
		timeout := time.Duration(cs.timeout) * time.Second
		timer := time.NewTimer(timeout)
		select {
		case <-cs.status:
			timer.Stop()
			wg.Done()
			return
		case <-timer.C:
			alive = false
			Log.Errorf("No response after %d seconds, timeout!", cs.timeout)
		}
	}
}

func newResponse(t interface{}) Response {
	ct1, _ := t.(float64)
	t1 := ResponseType(int64(ct1))
	if t1 == BASIC_R {
		return &BasicResponse{}
	} else if t1 == HTTP_R {
		return &HttpResponse{}
	}
	return nil
}

func (qc *QuicConnection) ListenToClient() {
	for {
		if b, err := NewReader(qc.Stream).Read(); err != nil {
			Log.Errorf("Error: %v", err)
			qc.Cancel()
			break
		} else {
			request := map[string]interface{}{}
			if err := json.Unmarshal(b, &request); err != nil {
				fmt.Printf("%s\n", fmt.Sprintf("Found error %s when trying to unmarshal data from client %d.", qc.Stream.StreamID()))
			} else {
				if code, rt := request["Code"], request["ResponseType"];  code != nil && rt!= nil{
					response := newResponse(rt)
					if err := json.Unmarshal(b, &response); err != nil {
						Log.Errorf("It's not a valid command response packet: %v", err)
					} else {
						qc.onCommandResponse(response)
					}
					continue
				} else if ct := request["CType"]; ct != nil {
					//Logic for client registration
					ct1, _ := ct.(float64)
					if CmdType(int(ct1)) == REGISTER {
						cmd := BasicCommand{}
						e := json.Unmarshal(b, &cmd)
						if e != nil {
							Log.Errorf("It's not a valid register command packet: %v", e)
						}
						if resp := cmd.Validate(); resp == nil {
							resp := BasicResponse{
								Identifier:  cmd.Identifier,
								Code:        OK,
								Description: "The client is registered successfully.",
							}
							GetManager().AddConn(cmd.Identifier, qc)
							if e = qc.sendResponse(resp); e != nil {
								Log.Errorf("Error: %v", e)
							}
						} else {
							if e = qc.sendResponse(*resp); e != nil {
								Log.Errorf("Error: %v", e)
							}
						}
						continue
					}
				}
				Log.Errorf("Unknown packet %s", b)
			}
		}
	}
}

func (qc *QuicConnection) onCommandResponse(response Response) {
	if e := response.Validate(); e != nil {
		Log.Errorf("%s", e)
		return
	}
	if status := qc.commandStatus[response.GetSequence()]; status == nil {
		logrus.Errorf("Cannot find related command status for %d.", response.GetSequence())
		return
	} else {
		status.response = response
		status.status <- 1
	}
}

func (qc *QuicConnection) SendCommand(cmd Command) (Response, error) {
	var err1 error

	var wg sync.WaitGroup
	wg.Add(1)
	cs := commandStatus{
		status:   make(chan int),
		timeout:  10,
	}
	if qc.commandStatus == nil {
		qc.commandStatus = make(map[int]*commandStatus)
	}
	qc.commandStatus[cmd.GetSequence()] = &cs
	go cs.start(&wg)

	j := cmd.Json()
	if _, err := NewWriter(qc.Stream).Write(j); err != nil {
		err1 = err
	} else {
		Log.Infof("The command %s is sent successfully", j)
	}
	if err1 != nil {
		wg.Done()
		return nil, err1
	} else {
		wg.Wait()
		return cs.response, cs.error
	}
}

func (qc *QuicConnection) sendResponse(resp BasicResponse) error {
	j := resp.Json()
	if l, err := NewWriter(qc.Stream).Write(j); err != nil {
		return err
	} else {
		Log.Infof("The response %s is issued successfully with len %d", j, l)
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

type sequenceIDGenerator struct {
	sequence int
	mu sync.Mutex
}

var generator = &sequenceIDGenerator{}

func (sq *sequenceIDGenerator) inc() int{
	sq.mu.Lock()
	defer sq.mu.Unlock()
	sq.sequence++
	return sq.sequence
}

func GetNextId() int {
	return generator.inc()
}