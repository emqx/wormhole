package server

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	quic "github.com/lucas-clemente/quic-go"
	"math/big"
	"quicdemo/common"
	"time"
)

type WormholeServer struct {
	BindAddr string
}

// Start a rest that echos all data on the first stream opened by the internal
func (ws *WormholeServer) Start() {
	listener, err := quic.ListenAddr(ws.BindAddr, generateTLSConfig(), &quic.Config{KeepAlive:true, HandshakeTimeout: 10 * time.Second})
	if err != nil {
		fmt.Println(err)
		return
	}
	for {
		context, cancel := context.WithCancel(context.Background())

		sess, err := listener.Accept(context)
		if err != nil {
			fmt.Println(err)
			return
		}
		go func() {
			gstream, err := sess.AcceptStream(context)
			fmt.Println("Accepted stream.")
			if err != nil {
				panic(err)
			}
			conn := common.QuicConnection{
				Session: sess,
				Stream:  gstream,
				Cancel:  cancel,
			}
			conn.ListenToClient()
		}()
	}
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
