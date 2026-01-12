package app

import "fmt"

// Handler interface for HTTP handlers
type Handler interface {
	Handle(path string)
	GetPriority() int
}

// HomeHandler handles the home page
type HomeHandler struct {
	priority int
}

func NewHomeHandler() *HomeHandler {
	return &HomeHandler{priority: 100}
}

func (h *HomeHandler) Handle(path string) {
	fmt.Printf("[HomeHandler priority=%d] Handling: %s\n", h.priority, path)
}

func (h *HomeHandler) GetPriority() int {
	return h.priority
}

// APIHandler handles API requests
type APIHandler struct {
	priority int
}

func NewAPIHandler() *APIHandler {
	return &APIHandler{priority: 200}
}

func (h *APIHandler) Handle(path string) {
	fmt.Printf("[APIHandler priority=%d] Handling: %s\n", h.priority, path)
}

func (h *APIHandler) GetPriority() int {
	return h.priority
}

// AdminHandler handles admin requests
type AdminHandler struct {
	priority int
}

func NewAdminHandler() *AdminHandler {
	return &AdminHandler{priority: 50}
}

func (h *AdminHandler) Handle(path string) {
	fmt.Printf("[AdminHandler priority=%d] Handling: %s\n", h.priority, path)
}

func (h *AdminHandler) GetPriority() int {
	return h.priority
}

// Server registers and dispatches to handlers
type Server struct {
	handlers []Handler
	prefix   string
}

// NewServer creates a server with variadic handlers
// This demonstrates the target of the spread operator
func NewServer(handlers ...Handler) *Server {
	return &Server{handlers: handlers}
}

func (s *Server) HandleRequest(path string) {
	fmt.Printf("\nServer handling request: %s\n", path)
	for _, h := range s.handlers {
		h.Handle(path)
	}
}

func (s *Server) ShowHandlers() {
	fmt.Printf("\nRegistered %d handlers:\n", len(s.handlers))
	for i, h := range s.handlers {
		fmt.Printf("  %d. Priority: %d\n", i+1, h.GetPriority())
	}
}

// PrefixedServer demonstrates mixing regular args with spread
type PrefixedServer struct {
	handlers []Handler
	prefix   string
}

// NewPrefixedServer takes a prefix and variadic handlers
// Demonstrates mixing regular arguments with spread
func NewPrefixedServer(prefix string, handlers ...Handler) *PrefixedServer {
	return &PrefixedServer{handlers: handlers, prefix: prefix}
}

func (s *PrefixedServer) HandleRequest(path string) {
	fullPath := s.prefix + path
	fmt.Printf("\nPrefixedServer handling request: %s\n", fullPath)
	for _, h := range s.handlers {
		h.Handle(fullPath)
	}
}

func (s *PrefixedServer) ShowHandlers() {
	fmt.Printf("\nPrefixedServer (prefix=%s) has %d handlers:\n", s.prefix, len(s.handlers))
	for i, h := range s.handlers {
		fmt.Printf("  %d. Priority: %d\n", i+1, h.GetPriority())
	}
}

// GetAllHandlers returns a slice of handlers
// Used to demonstrate spreading service references
func GetAllHandlers(home, api, admin Handler) []Handler {
	return []Handler{home, api, admin}
}
