package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockDispatcher struct {
	lastMethod string
	lastParams []json.RawMessage
	listener   func(string, ...interface{})
}

func (m *mockDispatcher) DispatchRPC(method string, params []json.RawMessage) (interface{}, error) {
	m.lastMethod = method
	m.lastParams = params
	if method == "echo" {
		if len(params) > 0 {
			var str string
			_ = json.Unmarshal(params[0], &str)
			return str, nil
		}
		return "ok", nil
	}
	if method == "fail" {
		return nil, fmt.Errorf("simulated failure")
	}
	return nil, fmt.Errorf("unknown method: %s", method)
}

func (m *mockDispatcher) SetEventListener(l func(eventName string, data ...interface{})) {
	m.listener = l
}

func TestServerHealth(t *testing.T) {
	srv := New(ServerConfig{Host: "127.0.0.1", Port: 8080}, &mockDispatcher{})
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if resp["status"] != "ok" || resp["app"] != "newscid" {
		t.Errorf("Unexpected health response: %v", resp)
	}
}

func TestServerRPC_Success(t *testing.T) {
	mock := &mockDispatcher{}
	srv := New(ServerConfig{Host: "127.0.0.1", Port: 8080}, mock)

	body := []byte(`{"id": 1, "method": "echo", "params": ["hello world"]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/rpc", bytes.NewReader(body))
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Code)
	}

	var resp RPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if resp.Error != "" {
		t.Errorf("Expected no error, got: %s", resp.Error)
	}
	if resp.Result != "hello world" {
		t.Errorf("Expected result 'hello world', got: %v", resp.Result)
	}
}

func TestServerRPC_Error(t *testing.T) {
	mock := &mockDispatcher{}
	srv := New(ServerConfig{Host: "127.0.0.1", Port: 8080}, mock)

	body := []byte(`{"id": 2, "method": "fail", "params": []}`)
	req := httptest.NewRequest(http.MethodPost, "/api/rpc", bytes.NewReader(body))
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Code)
	}

	var resp RPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if resp.Error != "simulated failure" {
		t.Errorf("Expected error 'simulated failure', got: %s", resp.Error)
	}
}

func TestServerCORS(t *testing.T) {
	srv := New(ServerConfig{Host: "127.0.0.1", Port: 8080}, &mockDispatcher{})
	req := httptest.NewRequest(http.MethodOptions, "/api/rpc", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for OPTIONS, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Expected CORS header *, got: %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}
