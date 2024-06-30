package server

import "sync"

type Registry struct {
	servers map[string]*Server
	mu      sync.RWMutex
}

func NewRegistry() *Registry {
	return &Registry{
		servers: map[string]*Server{},
	}
}

func (r *Registry) AddServer(identifier string, server *Server) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.servers[identifier] = server
}

func (r *Registry) GetServer(identifier string) *Server {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.servers[identifier]
}

func (r *Registry) RemoveServer(identifier string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.servers, identifier)
}

func (r *Registry) GetServers() []*Server {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var servers []*Server
	for _, s := range r.servers {
		servers = append(servers, s)
	}
	return servers
}
