package rest

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jinfahua/wormhole/common"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	ContentType     = "Content-Type"
	ContentTypeJSON = "application/json"
)

func jsonResponse(i interface{}, w http.ResponseWriter) {
	w.Header().Add(ContentType, ContentTypeJSON)
	enc := json.NewEncoder(w)
	err := enc.Encode(i)
	// Problems encoding
	if err != nil {
		handleError(w, err, "")
		return
	}
}

// Handle applies the specified error and error concept tot he HTTP response writer
func handleError(w http.ResponseWriter, err error, prefix string) {
	message := prefix
	if message != "" {
		message += ": "
	}
	message += err.Error()
	common.Log.Error(message)
	var ec = http.StatusBadRequest
	http.Error(w, message, ec)
}

func register(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	node := common.Agent{}
	err := json.NewDecoder(req.Body).Decode(&node)
	if err != nil {
		handleError(w, err, "")
	}
	if n, err := common.NewNodeMemCache().Add(node); err != nil {
		handleError(w, err, "")
	} else {
		jsonResponse(n, w)
	}
}

type OK struct {
	Message string
}

func delete(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	vars := mux.Vars(req)
	id := vars["id"]
	if err := common.NewNodeMemCache().DeleteById(id); err != nil {
		handleError(w, err, "")
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("%s is deleted.", id)))
	}
}

func update(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	node := common.Agent{}
	err := json.NewDecoder(req.Body).Decode(&node)
	if err != nil {
		handleError(w, err, "")
	}
	if n, err := common.NewNodeMemCache().Update(node); err != nil {
		handleError(w, err, "")
	} else {
		jsonResponse(n, w)
	}
}

func list(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	if nodes, err := common.NewNodeMemCache().List(); err != nil {
		handleError(w, err, "")
	} else {
		jsonResponse(nodes, w)
	}
}

func processRequest(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	vars := mux.Vars(req)

	id := vars["id"]
	mware := vars["mware"]
	rest := vars["rest"]

	node := common.NewNodeMemCache().Cache[id]
	if node == nil {
		handleError(w, fmt.Errorf("The specified node %s cannot be found.", id), "")
		return
	}

	ware, err := common.NewMWMemoryCache().GetByName(id, mware)
	if err != nil {
		handleError(w, fmt.Errorf("The specified middleware %s in node %s cannot be found.", mware, id), "")
	}

	conn := common.GetManager().GetConn(id)
	if conn == nil {
		handleError(w, fmt.Errorf("The connection to node %s is not existed.", id), "")
		return
	}

	body, err := ioutil.ReadAll(req.Body)

	cmd := common.HttpCommand{
		BasicCommand: common.BasicCommand{
			Identifier: id,
			Sequence:   common.GetNextId(),
			CType:      common.HTTP,
		},
		HttpRequest: common.HttpRequest{
			Method:   req.Method,
			Host:     "",
			Port:     ware.Port,
			BasePath: ware.Path,
			Path:     rest,
			Headers:  req.Header,
			Body:     body,
		},
	}
	if resp, err := conn.SendCommand(&cmd); err != nil {
		handleError(w, fmt.Errorf("Found error %s when trying to issue command to node %s.", err, id), "")
	} else {
		if resp.GetResponseCode() != common.OK {
			handleError(w, fmt.Errorf("Found error %s when trying to get command result for node %s.", resp.GetDescription(), id), "")
			return
		}
		if hr, ok := resp.(*common.HttpResponse); ok {
			if hr.HttpResponseCode != 200 {
				http.Error(w, hr.HttpResponseText, hr.HttpResponseCode)
			}
			if hr.Header != nil {
				for k, _ := range hr.Header {
					w.Header().Set(k, hr.Header.Get(k))
				}
			}
			if hr.Body != nil {
				w.Write(hr.Body)
				return
			}
		} else {
			handleError(w, fmt.Errorf("Not a valid http-response when get command result for node %s.", id), "")
		}
	}
}

func mlist(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	vars := mux.Vars(req)
	id := vars["id"]
	if nodes, err := common.NewMWMemoryCache().List(id); err != nil {
		handleError(w, err, "")
	} else {
		jsonResponse(nodes, w)
	}
}

func mupdate(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	vars := mux.Vars(req)
	id := vars["id"]

	mw := common.Middleware{}
	err := json.NewDecoder(req.Body).Decode(&mw)
	if err != nil {
		handleError(w, err, "")
	}
	if n, err := common.NewMWMemoryCache().Update(id, mw); err != nil {
		handleError(w, err, "")
	} else {
		jsonResponse(n, w)
	}
}

func mregister(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	vars := mux.Vars(req)
	id := vars["id"]

	mware := common.Middleware{}
	err := json.NewDecoder(req.Body).Decode(&mware)
	if err != nil {
		handleError(w, err, "")
	}
	if n, err := common.NewMWMemoryCache().Add(id, mware); err != nil {
		handleError(w, err, "")
	} else {
		jsonResponse(n, w)
	}
}

func mdelete(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	vars := mux.Vars(req)
	id := vars["id"]
	name := vars["name"]
	if err := common.NewMWMemoryCache().DeleteByName(id, name); err != nil {
		handleError(w, err, "")
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("%s under node %s is deleted.", name, id)))
	}
}

func CreateRestServer(srv string, port int) *http.Server {
	r := mux.NewRouter()

	r.HandleFunc("/nodes/register", register).Methods(http.MethodPost)
	r.HandleFunc("/nodes/{id}", delete).Methods(http.MethodDelete)
	r.HandleFunc("/nodes/", update).Methods(http.MethodPut)
	r.HandleFunc("/nodes/", list).Methods(http.MethodGet)

	r.HandleFunc("/nodes/{id}/mware", mlist).Methods(http.MethodGet)
	r.HandleFunc("/nodes/{id}/mware", mregister).Methods(http.MethodPost)
	r.HandleFunc("/nodes/{id}/mware", mupdate).Methods(http.MethodPut)
	r.HandleFunc("/nodes/{id}/mware/{name}", mdelete).Methods(http.MethodDelete)

	r.HandleFunc("/wh/{id}/{mware}/{rest:[a-zA-Z0-9_=\\-\\/@\\.:%\\+~#\\?&]+}", processRequest).Methods(http.MethodPost, http.MethodGet, http.MethodDelete, http.MethodPut)

	server := &http.Server{
		Addr: fmt.Sprintf("%s:%d", srv, port),
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 60 * 5,
		ReadTimeout:  time.Second * 60 * 5,
		IdleTimeout:  time.Second * 60,
		Handler:      handlers.CORS(handlers.AllowedHeaders([]string{"Accept", "Accept-Language", "Content-Type", "Content-Language", "Origin"}))(r),
	}
	server.SetKeepAlivesEnabled(false)
	return server
}
