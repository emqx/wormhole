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
	"net/http"
	"os"
	"os/signal"
	"quicdemo/common"
	"quicdemo/rest"
	"syscall"
	"time"
)

type WormholeServer struct {
	BindAddr string
}

func NewServer() {
	conf, ok := common.GetSrvConf()
	if !ok {
		fmt.Println("Failed to init configuration, exiting...")
		return
	}

	ws := &WormholeServer{fmt.Sprintf("%s:%d", conf.Basic.BindAddr, conf.Basic.BindPort)}
	go func() { ws.Start() }()

	if conf.Rest.EnableRest {
		//Start rest service
		srvRest := rest.CreateRestServer(conf.Rest.RestBindAddr, conf.Rest.RestBindPort)
		go func() {
			var err error
			err = srvRest.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				common.Log.Fatal("Error serving rest service: ", err)
				os.Exit(1)
			}
		}()
	}

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
	<-sigint
	os.Exit(0)
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
			//fmt.Println("Accepted stream.")
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
		NextProtos:   []string{"emqx-wormhole"},
	}
}
