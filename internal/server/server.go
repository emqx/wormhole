package server

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	quic "github.com/lucas-clemente/quic-go"
	"io"
	"math/big"
	"quicdemo/common"
)

type WormholeServer struct {
	BindAddr string
	ListenMgr  common.Subject
}



func (ws *WormholeServer) OnEvent(t common.EventType, payload []byte) error {
	if t == common.RESPONSE {
		common.Log.Printf("Get response from rest %s.", string(payload))
		return nil
	} else if t == common.COMMAND {
		cmd := common.Command{}
		err := json.Unmarshal(payload, &cmd)
		if err != nil {
			return err
		}
		common.Log.Printf("Get command from rest %s.", string(payload))
		return ws.IssueResponse(common.Response{
			Identifier: cmd.Identifier,
			Code:       common.OK,
			Contents:   "Successful",
		})
	} else {
		return fmt.Errorf("Found error %s", string(payload))
	}
	
}
// Start a rest that echos all data on the first stream opened by the internal
func (ws *WormholeServer) Start() {
	listener, err := quic.ListenAddr(ws.BindAddr, generateTLSConfig(), nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	for {
		sess, err := listener.Accept(context.Background())
		if err != nil {
			fmt.Println(err)
			return
		}
		go func() {
			gstream, err := sess.AcceptStream(context.Background())
			fmt.Println("Accepted stream.")
			if err != nil {
				panic(err)
			}
			conn := common.QuicConnection{
				Session: sess,
				Stream:  gstream,
			}
			ws.ListenToClient(&conn)
		}()
	}
}

func (ws *WormholeServer) ListenToClient(conn *common.QuicConnection) {
	for {
		if b, err := common.NewReader(conn.Stream).Read(); err != nil {
			common.Log.Errorf("Error: %v", err)
		} else {
			fmt.Printf("Received %s\n", b)
			request := map[string]interface{}{}
			if err := json.Unmarshal(b, &request); err != nil {
				fmt.Printf("%d %s\n", common.ERROR, fmt.Sprintf("Found error %s when trying to unmarshal data from client %d.", err, conn.Stream.StreamID()))
			} else {
				if request["Code"] != nil {
					ws.ListenMgr.NotifyAll(common.RESPONSE, b)
				} else if ct := request["CType"]; ct != nil {
					//Logic for client registration
					ct1, _ := ct.(float64)
					if common.CmdType(int(ct1)) == common.REGISTER {
						cmd := common.Command{}
						e := json.Unmarshal(b, &cmd)
						if e != nil {
							common.Log.Errorf("Error: %v", e)
						}
						if cmd.Identifier != ""{
							resp := common.Response{
								Identifier:  cmd.Identifier,
								Code:        common.OK,
								Contents: "The client is registered successfully.",
							}
							common.GetManager().AddConn(cmd.Identifier, conn)
							if e = ws.IssueResponse(resp); e != nil {
								common.Log.Errorf("Error: %v", e)
							}
						} else {
							estr := fmt.Sprintf("Identifier is required for registration.")
							common.Log.Errorf("%s", estr)
							resp := common.Response{
								Code:        common.BAD_REQUEST,
								Contents: estr,
							}
							if e = ws.IssueResponse(resp); e != nil {
								common.Log.Errorf("Error: %v", e)
							}
						}
						continue
					}
					ws.ListenMgr.NotifyAll(common.COMMAND, b)
				} else {
					ws.ListenMgr.NotifyAll(common.UNKNOWN, b)
				}
			}
		}
	}
}

func (ws *WormholeServer) IssueCommand(cmd common.Command) error {
	if conn := common.GetManager()[cmd.Identifier]; &conn == nil {
		return fmt.Errorf("Cannot find connection for %s", cmd.Identifier)
	} else {
		j := cmd.Json()
		if _, err := common.NewWriter(conn.Stream).Write(j); err != nil {
			return err
		} else {
			common.Log.Infof("The command %s is issued successfully", j)
		}
	}
	return nil
}

func (ws *WormholeServer) IssueResponse(resp common.Response) error {
	if conn := common.GetManager()[resp.Identifier]; &conn == nil {
		return fmt.Errorf("Cannot find connection for %s", resp.Identifier)
	} else {
		j := resp.Json()
		if l, err := common.NewWriter(conn.Stream).Write(j); err != nil {
			return err
		} else {
			common.Log.Infof("The response %s is issued successfully with len %d", j, l)
		}
	}
	return nil
}

// A wrapper for io.Writer that also logs the message.
type loggingWriter struct {
	conn    common.QuicConnection
	session quic.Session
	io.Writer
}

// Setup a bare-bones TLS config for the rest
func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"quic-echo-example"},
	}
}
