package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"newscid/pkg/version"
)

// Dispatcher defines the interface for executing backend RPC methods and streaming events.
type Dispatcher interface {
	DispatchRPC(method string, params []json.RawMessage) (interface{}, error)
	SetEventListener(l func(eventName string, data ...interface{}))
}

// ServerConfig configures the HTTP / WebSocket server.
type ServerConfig struct {
	Host     string
	Port     int
	DataDir  string
	AssetsFS fs.FS
}

// RPCRequest represents an incoming JSON-RPC call.
type RPCRequest struct {
	ID     interface{}       `json:"id"`
	Method string            `json:"method"`
	Params []json.RawMessage `json:"params"`
}

// RPCResponse represents the outcome of an RPC call.
type RPCResponse struct {
	ID     interface{} `json:"id"`
	Result interface{} `json:"result,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// Server provides the standalone web/mobile HTTP server for NewScid.
type Server struct {
	config     ServerConfig
	dispatcher Dispatcher
	httpServer *http.Server
	listener   net.Listener
	mu         sync.RWMutex
	clients    map[chan []byte]bool
}

// New creates a new Server instance.
func New(config ServerConfig, dispatcher Dispatcher) *Server {
	if config.Host == "" {
		config.Host = "127.0.0.1"
	}
	if config.DataDir == "" {
		config.DataDir = filepath.Join(os.TempDir(), "newscid")
	}
	_ = os.MkdirAll(filepath.Join(config.DataDir, "uploads"), 0755)

	s := &Server{
		config:     config,
		dispatcher: dispatcher,
		clients:    make(map[chan []byte]bool),
	}

	if dispatcher != nil {
		dispatcher.SetEventListener(s.broadcastEvent)
	}

	return s
}

// broadcastEvent sends an SSE event to all connected web clients.
func (s *Server) broadcastEvent(eventName string, data ...interface{}) {
	payload := map[string]interface{}{
		"name": eventName,
		"data": data,
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	for ch := range s.clients {
		select {
		case ch <- bytes:
		default:
		}
	}
}

// Handler returns the http.Handler for all routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/rpc", s.handleRPC)
	mux.HandleFunc("/api/events", s.handleEvents)
	mux.HandleFunc("/api/upload", s.handleUpload)

	// Static assets handler
	var staticHandler http.Handler
	if s.config.AssetsFS != nil {
		staticHandler = s.spaHandler(s.config.AssetsFS)
	} else {
		staticHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("NewScid Server running (no static assets embedded). Use /api/health or /api/rpc."))
		})
	}
	mux.Handle("/", staticHandler)

	return s.corsMiddleware(mux)
}

// spaHandler serves single-page app static files, falling back to index.html for navigation.
func (s *Server) spaHandler(assetsFS fs.FS) http.Handler {
	// Look for sub-folder "frontend/dist" or root
	sub, err := fs.Sub(assetsFS, "frontend/dist")
	if err == nil {
		assetsFS = sub
	}

	fileServer := http.FileServer(http.FS(assetsFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		f, err := assetsFS.Open(path)
		if err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// Fallback to index.html for SPA client-side routes
		indexFile, err := assetsFS.Open("index.html")
		if err == nil {
			_ = indexFile.Close()
			r.URL.Path = "/index.html"
			fileServer.ServeHTTP(w, r)
			return
		}

		http.NotFound(w, r)
	})
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"app":     "newscid",
		"version": version.Short(),
		"commit":  version.GitCommit,
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(RPCResponse{
			ID:    req.ID,
			Error: fmt.Sprintf("invalid JSON request body: %v", err),
		})
		return
	}

	if s.dispatcher == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(RPCResponse{
			ID:    req.ID,
			Error: "no backend dispatcher configured",
		})
		return
	}

	result, err := s.dispatcher.DispatchRPC(req.Method, req.Params)
	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		_ = json.NewEncoder(w).Encode(RPCResponse{
			ID:    req.ID,
			Error: err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(RPCResponse{
		ID:     req.ID,
		Result: result,
	})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan []byte, 32)
	s.mu.Lock()
	s.clients[ch] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, ch)
		close(ch)
		s.mu.Unlock()
	}()

	// Initial ping
	_, _ = fmt.Fprintf(w, "event: connected\ndata: {}\n\n")
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			_, _ = fmt.Fprintf(w, "event: wails\ndata: %s\n\n", string(msg))
			flusher.Flush()
		}
	}
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// 100 MB max file upload
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		http.Error(w, "File too large or invalid multipart form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Missing file parameter in form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	destDir := filepath.Join(s.config.DataDir, "uploads")
	_ = os.MkdirAll(destDir, 0755)
	destPath := filepath.Join(destDir, filepath.Base(header.Filename))

	out, err := os.Create(destPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to save file: %v", err), http.StatusInternalServerError)
		return
	}
	defer out.Close()

	written, err := io.Copy(out, file)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed writing file: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"filePath": destPath,
		"fileName": header.Filename,
		"size":     written,
	})
}

// Start begins listening and serving HTTP requests.
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	s.listener = listener

	s.httpServer = &http.Server{
		Handler:      s.Handler(),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return s.httpServer.Serve(listener)
}

// Port returns the actual listening port (useful when Port was set to 0).
func (s *Server) Port() int {
	if s.listener == nil {
		return s.config.Port
	}
	return s.listener.Addr().(*net.TCPAddr).Port
}

// Addr returns the complete host:port address string.
func (s *Server) Addr() string {
	if s.listener == nil {
		return fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	}
	return s.listener.Addr().String()
}

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}
