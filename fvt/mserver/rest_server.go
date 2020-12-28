package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"net/http"
	"os"
	"os/signal"
	"quicdemo/rest"
	"syscall"
	"time"
)

func jsonResponse(i interface{}, w http.ResponseWriter) {
	w.Header().Add(rest.ContentType, rest.ContentTypeJSON)
	enc := json.NewEncoder(w)
	enc.Encode(i)
}

func getstream(w http.ResponseWriter, req *http.Request) {
	res := []string{"demo1", "demo2"}
	jsonResponse(res, w)
}

func createRestServer(port int) *http.Server {
	r := mux.NewRouter()

	r.HandleFunc("/streams", getstream).Methods(http.MethodGet)
	//r.HandleFunc("/nodes/{id}", delete).Methods(http.MethodDelete)
	//r.HandleFunc("/nodes/", update).Methods(http.MethodPut)
	//r.HandleFunc("/nodes/", list).Methods(http.MethodGet)


	server := &http.Server{
		Addr: fmt.Sprintf("0.0.0.0:%d", port),
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 60 * 5,
		ReadTimeout:  time.Second * 60 * 5,
		IdleTimeout:  time.Second * 60,
		Handler:      handlers.CORS(handlers.AllowedHeaders([]string{"Accept", "Accept-Language", "Content-Type", "Content-Language", "Origin"}))(r),
	}
	server.SetKeepAlivesEnabled(false)
	return server
}

func main() {
	srvRest := createRestServer(9081)
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
