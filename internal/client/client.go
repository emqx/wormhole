package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	quic "github.com/lucas-clemente/quic-go"
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

func (qcc *QCClient)OnEvent(t common.EventType, payload []byte) error {
	if t == common.RESPONSE {
		common.Log.Printf("Get response from rest %s.", string(payload))
	} else if t == common.COMMAND {
		common.Log.Printf("Get command from rest %s.", string(payload))
		return qcc.WriteTo(common.Response{
			Identifier:  qcc.Identifier,
			Code:        common.OK,
			Description: "Successful",
		})
	} else {
		return fmt.Errorf("Found error %s", string(payload))
	}
	return nil
}

// We start a rest echoing data on the first stream the internal opens,
// then connect with a internal, send the message, and wait for its receipt.
func main() {
	conf := common.GetConf()
	conf.InitLog()

	mgr := common.ListenerMgr{}
	qcc := QCClient{"127.0.0.1:4242", "test", nil, &mgr}
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


