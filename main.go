package main

import (
	"net/http"
	"os"
	"os/signal"
	"quicdemo/common"
	"quicdemo/internal/server"
	"quicdemo/rest"
	"syscall"
)

func main() {
	conf := common.GetConf()
	if !conf.InitLog() {
		return
	}
	ws := &server.WormholeServer{common.Addr}
	go func() { ws.Start() }()
	//Start rest service
	srvRest := rest.CreateRestServer(9999)
	go func() {
		var err error
		err = srvRest.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			//logger.Fatal("Error serving rest service: ", err)
		}
	}()

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
	<-sigint
	os.Exit(0)
}