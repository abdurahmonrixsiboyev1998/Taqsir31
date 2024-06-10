package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/ybbus/jsonrpc/v2"
)

type Storage interface {
	Get(key string) (string, error)
	Post(key, value string) error
	Put(key, value string) error
	Delete(key string) error
}

type InMemoryStorage struct {
	data map[string]string
	mu   sync.RWMutex
}

func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		data: make(map[string]string),
	}
}

func (s *InMemoryStorage) Get(key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.data[key]
	if !ok {
		return "", fmt.Errorf("key not found")
	}
	return value, nil
}

func (s *InMemoryStorage) Post(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
	return nil
}

func (s *InMemoryStorage) Put(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
	return nil
}

func (s *InMemoryStorage) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}

type JSONRPCServer struct {
	storage Storage
}

func NewJSONRPCServer(storage Storage) *JSONRPCServer {
	return &JSONRPCServer{
		storage: storage,
	}
}

func (s *JSONRPCServer) HandleRequest(w http.ResponseWriter, r *http.Request) {
	var req jsonrpc.RPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	var res jsonrpc.RPCResponse
	res.JSONRPC = "2.0"
	res.ID = req.ID

	switch req.Method {
	case "get":
		key := req.Params.(map[string]interface{})["key"].(string)
		value, err := s.storage.Get(key)
		if err != nil {
			res.Error = &jsonrpc.RPCError{Code: 1, Message: err.Error()}
		} else {
			res.Result = value
		}
	case "post":
		params := req.Params.(map[string]interface{})
		key := params["key"].(string)
		value := params["value"].(string)
		err := s.storage.Post(key, value)
		if err != nil {
			res.Error = &jsonrpc.RPCError{Code: 1, Message: err.Error()}
		} else {
			res.Result = "success"
		}
	case "put":
		params := req.Params.(map[string]interface{})
		key := params["key"].(string)
		value := params["value"].(string)
		err := s.storage.Put(key, value)
		if err != nil {
			res.Error = &jsonrpc.RPCError{Code: 1, Message: err.Error()}
		} else {
			res.Result = "success"
		}
	case "delete":
		key := req.Params.(map[string]interface{})["key"].(string)
		err := s.storage.Delete(key)
		if err != nil {
			res.Error = &jsonrpc.RPCError{Code: 1, Message: err.Error()}
		} else {
			res.Result = "success"
		}
	default:
		res.Error = &jsonrpc.RPCError{Code: -32601, Message: "Method not found"}
	}

	if err := json.NewEncoder(w).Encode(res); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// StartServer starts the JSON-RPC server
func StartServer(address string, storage Storage) {
	server := NewJSONRPCServer(storage)
	http.HandleFunc("/rpc", server.HandleRequest)
	log.Printf("Starting server on %s...", address)
	if err := http.ListenAndServe(address, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
