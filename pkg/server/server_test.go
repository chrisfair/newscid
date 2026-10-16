package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"
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

func TestServerRPCEdgeCases(t *testing.T) {
	// 1. Method Not Allowed (GET on /api/rpc)
	srv := New(ServerConfig{}, &mockDispatcher{})
	req := httptest.NewRequest(http.MethodGet, "/api/rpc", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Expected 405 Method Not Allowed, got %d", w.Code)
	}

	// 2. Invalid JSON request body
	reqBadJSON := httptest.NewRequest(http.MethodPost, "/api/rpc", bytes.NewReader([]byte("{invalid-json")))
	wBadJSON := httptest.NewRecorder()
	srv.Handler().ServeHTTP(wBadJSON, reqBadJSON)
	if wBadJSON.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request, got %d", wBadJSON.Code)
	}

	// 3. Nil Dispatcher
	srvNil := New(ServerConfig{}, nil)
	reqNil := httptest.NewRequest(http.MethodPost, "/api/rpc", bytes.NewReader([]byte(`{"id": 1, "method": "test"}`)))
	wNil := httptest.NewRecorder()
	srvNil.Handler().ServeHTTP(wNil, reqNil)
	if wNil.Code != http.StatusInternalServerError {
		t.Fatalf("Expected 500 Internal Server Error, got %d", wNil.Code)
	}
}

func TestServerUploadAndEvents(t *testing.T) {
	tmpDir := t.TempDir()
	mock := &mockDispatcher{}
	srv := New(ServerConfig{DataDir: tmpDir}, mock)

	// 1. GET /api/upload should fail (Method Not Allowed)
	reqGetUpload := httptest.NewRequest(http.MethodGet, "/api/upload", nil)
	wGetUpload := httptest.NewRecorder()
	srv.Handler().ServeHTTP(wGetUpload, reqGetUpload)
	if wGetUpload.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Expected 405, got %d", wGetUpload.Code)
	}

	// 2. Broadcast event
	mock.listener("testEvent", "sampleData")

	// 3. Fallback text response when AssetsFS is nil
	reqRoot := httptest.NewRequest(http.MethodGet, "/", nil)
	wRoot := httptest.NewRecorder()
	srv.Handler().ServeHTTP(wRoot, reqRoot)
	if wRoot.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", wRoot.Code)
	}
}

func TestServerLifecycleFileUploadAndSPA(t *testing.T) {
	tmpDir := t.TempDir()
	mock := &mockDispatcher{}

	// Mock SPA files
	mockFS := fstest.MapFS{
		"frontend/dist/index.html": &fstest.MapFile{
			Data: []byte("<html><body>NewScid App</body></html>"),
		},
		"frontend/dist/style.css": &fstest.MapFile{
			Data: []byte("body { background: #000; }"),
		},
	}

	srv := New(ServerConfig{
		Host:     "127.0.0.1",
		Port:     0, // random open port
		DataDir:  tmpDir,
		AssetsFS: mockFS,
	}, mock)

	// 1. Test SPA serving
	reqSPA := httptest.NewRequest(http.MethodGet, "/style.css", nil)
	wSPA := httptest.NewRecorder()
	srv.Handler().ServeHTTP(wSPA, reqSPA)
	if wSPA.Code != http.StatusOK {
		t.Fatalf("Expected 200 for /style.css, got %d", wSPA.Code)
	}

	// 2. Test SPA fallback to index.html
	reqFallback := httptest.NewRequest(http.MethodGet, "/random-route", nil)
	wFallback := httptest.NewRecorder()
	srv.Handler().ServeHTTP(wFallback, reqFallback)
	if wFallback.Code != http.StatusOK && wFallback.Code != http.StatusMovedPermanently {
		t.Fatalf("Expected 200 or 301 fallback for /random-route, got %d", wFallback.Code)
	}

	// 3. Test File Upload
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "test_game.pgn")
	if err != nil {
		t.Fatalf("CreateFormFile error: %v", err)
	}
	_, _ = part.Write([]byte(`[Event "Test"] 1. e4 e5 *`))
	_ = writer.Close()

	reqUpload := httptest.NewRequest(http.MethodPost, "/api/upload", &body)
	reqUpload.Header.Set("Content-Type", writer.FormDataContentType())
	wUpload := httptest.NewRecorder()
	srv.Handler().ServeHTTP(wUpload, reqUpload)
	if wUpload.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for file upload, got %d: %s", wUpload.Code, wUpload.Body.String())
	}

	// 4. Test Server Start & Stop
	go func() {
		_ = srv.Start()
	}()
	time.Sleep(50 * time.Millisecond)

	if srv.Port() == 0 {
		t.Errorf("Expected non-zero listening port")
	}
	if srv.Addr() == "" {
		t.Errorf("Expected non-empty Addr")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Stop(ctx)
}


