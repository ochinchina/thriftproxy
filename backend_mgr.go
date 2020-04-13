package main

import (
	"fmt"
	"sync"
)

var indexOutOfBoundError error = fmt.Errorf("Index out of bound")

type BackendMgr struct {
	sync.Mutex
	backends []Backend
}

// NewBackendMgr create a BackendMgr object
func NewBackendMgr() *BackendMgr {
	return &BackendMgr{backends: make([]Backend, 0)}
}

// Exists Check if the backend with address exists or not
func (b *BackendMgr) Exists(addr string) bool {
	b.Lock()
	defer b.Unlock()
	_, err := b.getIndex(addr)
	return err == nil
}

// Size get number of backends
func (b *BackendMgr) Size() int {
	b.Lock()
	defer b.Unlock()

	return len(b.backends)
}

// GetIndex get backend by index
func (b *BackendMgr) GetIndex(index int) (Backend, error) {
	b.Lock()
	defer b.Unlock()

	if index >= 0 && index < len(b.backends) {
		return b.backends[index], nil
	}
	return nil, indexOutOfBoundError
}

// Add add a backend
func (b *BackendMgr) Add(backend Backend) {
	b.Lock()
	defer b.Unlock()
	b.backends = append(b.backends, backend)
}

// Get get backend by address
func (b *BackendMgr) Get(addr string) (Backend, error) {
	b.Lock()
	defer b.Unlock()

	index, err := b.getIndex(addr)
	if err == nil {
		return b.backends[index], nil
	}
	return nil, err
}

// Remove remove backend by address
func (b *BackendMgr) Remove(addr string) (Backend, error) {
	b.Lock()
	defer b.Unlock()

	index, err := b.getIndex(addr)
	if err == nil {
		backend := b.backends[index]
		b.backends = append(b.backends[0:index], b.backends[index+1:]...)
		return backend, nil
	}
	return nil, err

}

// GetAll get all the backend
func (b *BackendMgr) GetAll() []Backend {
	b.Lock()
	defer b.Unlock()

	r := make([]Backend, 0)
	r = append(r, b.backends...)
	return r
}

// getIndex get the index of backend by address
func (b *BackendMgr) getIndex(addr string) (int, error) {
	for index, backend := range b.backends {
		if backend.GetAddr() == addr {
			return index, nil
		}
	}
	return -1, fmt.Errorf("No such backend %s", addr)
}
