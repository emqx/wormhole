package common

import (
	"fmt"
	"github.com/google/uuid"
	"net/http"
)

type Node struct {
	Name        string
	Identifier  string
	Description string
}

type Middleware struct {
	Name       string
	Path       string
	Port       int
}

type NodeManager interface {
	List() ([]Node, error)
	Add(node Node) (*Node, error)
	Update(node Node) (*Node, error)
	DeleteById(identifier string) (error)
}

type NodeMemoryCache struct {
	Cache map[string]*Node
}

var nmCache *NodeMemoryCache

func NewNodeMemCache() *NodeMemoryCache{
	if nmCache == nil {
		nmCache = &NodeMemoryCache{}
		nmCache.Cache = make(map[string]*Node)
	}
	return nmCache
}

func (n *Node)validate() bool {
	if n.Name == "" {
		return false
	}
	return true
}

func (nc *NodeMemoryCache)List() ([]Node, error) {
	mwares := make([]Node, len(nc.Cache))
	for _, v := range nc.Cache {
		mwares = append(mwares, *v)
	}
	return mwares, nil
}

func (nc *NodeMemoryCache)Add(n Node) (*Node, error) {
	uuid, _ := uuid.NewUUID()
	n.Identifier = uuid.String()
	nc.Cache[n.Identifier] = &n
	return &Node{}, nil
}

func (nc *NodeMemoryCache)Update(n Node) (*Node, error) {
	if !n.validate() {
		return nil, fmt.Errorf("Not valid node settings %v", n)
	}
	if n.Identifier == "" {
		return nil, fmt.Errorf("Identifier is expected %v", n)
	}
	nc.Cache[n.Identifier] = &n
	return &n, nil
}

func (nc *NodeMemoryCache) DeleteById(id string) (error) {
	if id == "" {
		return fmt.Errorf("id %s cannot be empty", id)
	}
	delete(nc.Cache, id)
	return nil
}

type Middlewares []Middleware

type MiddlewareManager interface {
	List(nodeid string) (Middlewares, error)
	Add(nodeid string, middleware Middleware) (*Middleware, error)
	Update(nodeid string, middleware Middleware) (*Middleware, error)
	DeleteByName(nodeid string, name string) (error)
	GetByName(nodeid string, name string) (error)
}

func (mws *Middlewares) GetMiddlewareByName(name string) (*Middleware) {
	for _, mw := range *mws {
		if mw.Name == name {
			return &mw
		}
	}
	return nil
}

type MWMemoryCache struct {
	Cache map[string]Middlewares
}

func (mc *MWMemoryCache) List(nodeid string) ([]Middleware, error) {
	mws := mc.Cache[nodeid]
	if mws == nil {
		return nil, fmt.Errorf("Cannot find middlewares for id %s", nodeid)
	}
	mwares := make(Middlewares, len(mws))
	for _, v := range mws {
		mwares = append(mwares, v)
	}
	return mwares, nil
}

func (mc *MWMemoryCache) Add(nodeid string, m Middleware) (*Middleware, error) {
	if !m.validateMiddleware() {
		return nil, fmt.Errorf("Not valid middleware settings %v", m)
	}
	mws := mc.Cache[nodeid]
	if mws == nil {
		mws = make(Middlewares, 10)
	}
	mws = append(mws, m)
	mc.Cache[nodeid] = mws
	return &m, nil
}

func (mc *Middleware) validateMiddleware() (bool) {
	if mc.Name == "" || mc.Path == "" || mc.Port == 0 {
		return false
	}
	return true
}

func (mc *MWMemoryCache) Update(nodeid string, m Middleware) (*Middleware, error) {
	if !m.validateMiddleware() {
		return nil, fmt.Errorf("Not valid middleware settings %v", m)
	}
	mws := mc.Cache[nodeid]
	if mws == nil {
		return nil, fmt.Errorf("Cannot find middlewares for id %s", nodeid)
	}

	index := -1
	for idx, mw := range mws {
		if mw.Name == m.Name {
			index = idx
		}
	}
	if index != -1 {
		mc.Cache[nodeid] = removeMware(mws, index)
		return &m, nil
	} else {
		return nil, fmt.Errorf("Cannot find the middleware with name %s", m.Name)
	}
}

func removeMware(middlewares Middlewares, s int) Middlewares {
	return append(middlewares[:s], middlewares[s+1:]...)
}

func (mc *MWMemoryCache) DeleteByName(nodeid string, name string) (error) {
	if nodeid == "" || name == "" {
		return fmt.Errorf("nodeid or name cannot be empty ")
	}
	mws := mc.Cache[nodeid]
	if mws == nil {
		return fmt.Errorf("Cannot find middlewares for id %s", nodeid)
	}
	index := -1

	for idx, mw := range mws {
		if mw.Name == name {
			index = idx
		}
	}
	if index != -1 {
		mc.Cache[nodeid] = removeMware(mws, index)
		return nil
	} else {
		return fmt.Errorf("Cannot find the middleware with name %s", name)
	}
}

func (mc *MWMemoryCache) GetByName(nodeid string, name string) (*Middleware, error) {
	if nodeid == "" || name == "" {
		return nil, fmt.Errorf("nodeid or name cannot be empty ")
	}
	mws := mc.Cache[nodeid]
	if mws == nil {
		return nil, fmt.Errorf("Cannot find middlewares for id %s", nodeid)
	}

	for _, mw := range mws {
		if mw.Name == name {
			return &mw, nil
		}
	}
	return nil, fmt.Errorf("Cannot find the middleware with name %s", name)
}

var memCache *MWMemoryCache

func NewMWMemoryCache() *MWMemoryCache {
	if memCache == nil {
		memCache = &MWMemoryCache{}
		memCache.Cache = map[string]Middlewares{}
	}
	return memCache
}

type HttpRequest struct {
	Host    string
	Port    int
	BasePath string
	Path    string
	Headers http.Header
	Body    []byte
}

type HttpResponse struct {
	Headers map[string]string
	Body    []byte
}

