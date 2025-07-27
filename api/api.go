package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"emailchecker"
)

type Server struct {
	checker *emailchecker.EmailChecker
	port    string
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

func NewServer(checker *emailchecker.EmailChecker, port string) *Server {
	return &Server{
		checker: checker,
		port:    port,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/check", s.handleEmailCheck)
	mux.HandleFunc("/health", s.handleHealth)

	server := &http.Server{
		Addr:           ":" + s.port,
		Handler:        mux,
		ReadTimeout:    90 * time.Second,
		WriteTimeout:   90 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	log.Printf("Starting email checker API server on port %s", s.port)
	return server.ListenAndServe()
}

func (s *Server) handleEmailCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	email := r.URL.Query().Get("email")
	if email == "" {
		s.writeError(w, http.StatusBadRequest, "email parameter is required")
		return
	}

	ctx := r.Context()
	params := emailchecker.EmailCheckParams{
		Email: email,
	}

	result, err := s.checker.Check(ctx, params)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to check email: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	}

	if err := json.NewEncoder(w).Encode(errorResp); err != nil {
		log.Printf("Failed to encode error response: %v", err)
	}
}
