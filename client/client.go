package client

import (
	//"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/jinfahua/wormhole/common"
	quic "github.com/lucas-clemente/quic-go"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type QCClient struct {
	Server     string
	Identifier string
	Stream     quic.Stream
	cancel     context.CancelFunc
}

func NewClient() {
	conf, ok := common.GetAgentConf()
	if !ok {
		fmt.Println("Failed to init configuration, exiting...")
		return
	}

	args := os.Args[2:]
	if len(args) == 0 {
		fmt.Printf("The node identifier is expected.")
		os.Exit(0)
	}

	common.Log.Printf("The node identifier is %s\n", args[0])
	qcc := QCClient{Server: fmt.Sprintf("%s:%d", conf.Basic.Server, conf.Basic.Port), Identifier: args[0]}

	err := qcc.clientMain()
	if err != nil {
		panic(err)
	}

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
	<-sigint
	qcc.cancel()
	os.Exit(0)
}

func (qcc *QCClient) sendRequest(r common.HttpRequest) (*http.Response, error) {
	common.Log.Debugf("URL is: %s", r.ToString())
	//if req, error := http.NewRequest(r.Method, r.ToString(), bytes.NewBuffer(r.Body)); error != nil {
	if req, error := http.NewRequest(r.Method, r.ToString(), r.Body); error != nil {
		common.Log.Errorf("Find error %s when producing request %v.", error, r)
		return nil, error
	} else {
		req.Header = r.Headers
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		client := &http.Client{}
		return client.Do(req)
	}
}

func (qcc *QCClient) clientMain() error {
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"emqx-wormhole"},
	}

	session, err := quic.DialAddr(qcc.Server, tlsConf, &quic.Config{KeepAlive: true, HandshakeTimeout: 10 * time.Second})
	if err != nil {
		return err
	}

	context, cancel := context.WithCancel(context.Background())
	qcc.cancel = cancel
	stream, err := session.OpenStreamSync(context)
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

func (qcc *QCClient) onCommand(cmd *common.HttpCommand) error {
	//fmt.Printf("%T - %v", cmd.Payload, cmd)
	if response, err1 := qcc.sendRequest(cmd.HttpRequest); err1 != nil {
		return qcc.WriteTo(common.BasicResponse{
			Identifier:   qcc.Identifier,
			ResponseType: common.BASIC_R,
			Sequence:     cmd.Sequence,
			Code:         common.ERROR_FOUND,
			Description:  err1.Error(),
		})
	} else {
		common.Log.Debugf("headers from remote server %v", response.Header)
		if c, e := getContent(*response); e != nil {
			return qcc.WriteTo(common.BasicResponse{
				Identifier:   qcc.Identifier,
				ResponseType: common.BASIC_R,
				Sequence:     cmd.Sequence,
				Code:         common.ERROR_FOUND,
				Description:  e.Error(),
			})
		} else {
			return qcc.WriteTo(
				common.HttpResponse{
					BasicResponse: common.BasicResponse{
						ResponseType: common.HTTP_R,
						Identifier:   qcc.Identifier,
						Sequence:     cmd.Sequence,
						Code:         common.OK,
					},
					Header:           response.Header,
					HttpResponseCode: response.StatusCode,
					HttpResponseText: response.Status,
					Body:             c,
				})
		}
	}
	return nil
}

func getContent(resp http.Response) ([]byte, error) {
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func (qcc *QCClient) onResponse(response *common.BasicResponse) {
	common.Log.Printf("Get response from rest %s.", response)
}

func (qcc *QCClient) ListenToSrv() {
	for {
		if rawData, err := common.NewReader(qcc.Stream).Read(); err != nil {
			qcc.cancel()
			break
		} else {
			result := map[string]interface{}{}
			if e := json.Unmarshal(rawData, &result); e != nil {
				common.Log.Errorf("Found error when trying to unmarshal data from server %s", rawData)
			} else {
				if result["Code"] != nil {
					response := common.BasicResponse{}
					err := json.Unmarshal(rawData, &response)
					if err != nil {
						common.Log.Errorf("Invalid response packet from server %s", err)
					} else {
						qcc.onResponse(&response)
					}
					continue
				} else if t := result["CType"]; t != nil {
					common.Log.Debugf("%s", rawData)
					t1, _ := t.(float64)
					if common.HTTP == common.CmdType(int64(t1)) {
						hcmd := common.HttpCommand{}
						err := json.Unmarshal(rawData, &hcmd)
						if err != nil {
							common.Log.Errorf("Invalid packet from server %s", err)
						} else {
							if err := qcc.onCommand(&hcmd); err != nil {
								common.Log.Errorf("Failed to process command %s", err)
							}
						}
					} else {
						common.Log.Errorf("Not supported command type %d", common.CmdType(int64(t1)))
					}
					continue
				}
				common.Log.Errorf("Invalid result %s", rawData)
			}
		}
	}
}

func (qcc *QCClient) Register() error {
	cmd := common.BasicCommand{
		Identifier: qcc.Identifier,
		CType:      common.REGISTER,
	}
	if err := qcc.WriteTo(cmd); err != nil {
		return err
	}
	qcc.ListenToSrv()
	return nil
}
