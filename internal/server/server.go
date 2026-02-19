package server

import (
	"log"
	"net/http"
	"os"
)

// Server holds the configuration for the file server.
type Server struct {
	BasePath string
	Port     string
}

// New creates a new Server instance.
func New(basePath, port string) *Server {
	return &Server{
		BasePath: basePath,
		Port:     port,
	}
}

// Start initializes and starts the HTTP server.
func (s *Server) Start() error {
	if !dirExists(s.BasePath) {
		return os.ErrNotExist
	}

	mux := http.NewServeMux()

	// FileServer handler
	fs := http.FileServer(http.Dir(s.BasePath))

	// Wrap with logging middleware
	mux.Handle("/", loggingMiddleware(fs))

	bindTo := "0.0.0.0:" + s.Port
	return http.ListenAndServe(bindTo, mux)
}

// dirExists checks if a directory exists.
func dirExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// loggingMiddleware logs requests and sets headers.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[srv] info: [%s] %s %s", r.Host, r.Method, r.URL)
		w.Header().Set("Cache-Control", "max-age=5")
		next.ServeHTTP(w, r)
	})
}
