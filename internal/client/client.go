package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	quic "github.com/lucas-clemente/quic-go"
	"net/http"
	"os"
	"os/signal"
	"quicdemo/common"
	"syscall"
)


type QCClient struct {
	Server     string
	Identifier string
	Stream     quic.Stream
	ListenMgr  common.Subject
}

func (qcc *QCClient)sendRequest(r common.HttpRequest) (*http.Response, error){
	req, _ := http.NewRequest(r.Method, r.ToURL(), bytes.NewBuffer(r.Body))
	req.Header = r.Headers
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{}
	return client.Do(req)
}

func (qcc *QCClient)OnEvent(t common.EventType, payload []byte) error {
	if t == common.RESPONSE {
		common.Log.Printf("Get response from rest %s.", string(payload))
	} else if t == common.COMMAND {
		cmd := &common.Command{}
		err := json.Unmarshal(payload, &cmd);
		if err != nil {
			return err
		}
		//common.Log.Printf("Get command from rest %s.", string(payload))
		if cmd.CType == common.HTTP {
			if http, ok := cmd.Payload.(common.HttpRequest); ok {
				if response, err1 := qcc.sendRequest(http); err1 != nil {
					return qcc.WriteTo(common.Response{
						Identifier: qcc.Identifier,
						Code:       common.ERROR_FOUND,
						Contents:   err1.Error(),
					})
				} else {
					return qcc.WriteTo(common.Response{
						Identifier: qcc.Identifier,
						Code:       common.OK,
						Contents:   response,
					})
				}
				//fmt.Printf("%s", http.Method)
			} else {
				return fmt.Errorf("Not valid http request!")
			}
		}

	} else {
		return fmt.Errorf("Found error %s", string(payload))
	}
	return nil
}

// We start a rest echoing data on the first stream the internal opens,
// then connect with a internal, send the message, and wait for its receipt.
func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Printf("The node identifier is expected.")
		os.Exit(0)
	}

	conf := common.GetConf()
	conf.InitLog()

	mgr := common.ListenerMgr{}
	common.Log.Printf("The node identifier is %s\n", args[0])
	qcc := QCClient{"127.0.0.1:4242", args[0], nil, &mgr}
	mgr.Add(&qcc)

	err := qcc.clientMain()
	if err != nil {
		panic(err)
	}

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
	<-sigint
	os.Exit(0)
}

func (qcc *QCClient) clientMain() error {
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-echo-example"},
	}
	session, err := quic.DialAddr(qcc.Server, tlsConf, &quic.Config{KeepAlive:true})
	if err != nil {
		return err
	}

	stream, err := session.OpenStreamSync(context.Background())
	if err != nil {
		return err
	}
	qcc.Stream = stream
	if e := qcc.Register(); e != nil {
		return e
	}
	return nil
}

func (qcc *QCClient) WriteTo(con interface{}) error {
	j, e := json.Marshal(con)
	if e != nil {
		return e
	}
	if _, err := common.NewWriter(qcc.Stream).Write(j); err != nil {
		return fmt.Errorf("Found error when sending out request - %t", err)
	} else {
		common.Log.Infof("Request %s is sent out successfully. Waiting for the response.", j)
	}
	return nil
}

func (qcc *QCClient) ListenToSrv() {
	for {
		if rawData, err := common.NewReader(qcc.Stream).Read(); err != nil {
			qcc.ListenMgr.NotifyAll(common.ERROR, []byte("Found error when trying to get result from rest."))
		} else {
			//common.Log.Printf("From rest %s", rawData)
			result := map[string]interface{}{}
			if e := json.Unmarshal(rawData, &result); e != nil {
				qcc.ListenMgr.NotifyAll(common.ERROR, []byte(fmt.Sprintf("Found error when trying to unmarshal data from rest %s.", e)))
			} else {
				if result["Code"] != nil {
					qcc.ListenMgr.NotifyAll(common.RESPONSE, rawData)
				} else if result["CType"] != nil {
					qcc.ListenMgr.NotifyAll(common.COMMAND, rawData)
				} else {
					qcc.ListenMgr.NotifyAll(common.UNKNOWN, rawData)
				}
			}
		}
	}
}

func (qcc *QCClient) Register() error {
	cmd := common.Command{
		Identifier: qcc.Identifier,
		CType:      common.REGISTER,
		Payload:    nil,
	}
	if err := qcc.WriteTo(cmd); err != nil {
		return err
	}
	qcc.ListenToSrv()
	return nil
}


