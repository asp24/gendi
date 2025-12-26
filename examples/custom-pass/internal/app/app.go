package app

import (
	"fmt"
	"log/slog"
)

// Repository interfaces
type UserRepository interface {
	FindUser(id int) string
}

type ProductRepository interface {
	FindProduct(id int) string
}

// Repository implementations
type userRepo struct {
	dsn string
}

func NewUserRepository(dsn string) UserRepository {
	return &userRepo{dsn: dsn}
}

func (r *userRepo) FindUser(id int) string {
	return fmt.Sprintf("User #%d from %s", id, r.dsn)
}

type productRepo struct {
	dsn string
}

func NewProductRepository(dsn string) ProductRepository {
	return &productRepo{dsn: dsn}
}

func (r *productRepo) FindProduct(id int) string {
	return fmt.Sprintf("Product #%d from %s", id, r.dsn)
}

// HTTP Handler interface
type HTTPHandler interface {
	Handle() string
}

// User handler
type UserHandler struct {
	repo   UserRepository
	logger *slog.Logger
}

func NewUserHandler(repo UserRepository, logger *slog.Logger) HTTPHandler {
	return &UserHandler{repo: repo, logger: logger}
}

func (h *UserHandler) Handle() string {
	h.logger.Info("UserHandler: handling request")
	return h.repo.FindUser(42)
}

// Product handler
type ProductHandler struct {
	repo   ProductRepository
	logger *slog.Logger
}

func NewProductHandler(repo ProductRepository, logger *slog.Logger) HTTPHandler {
	return &ProductHandler{repo: repo, logger: logger}
}

func (h *ProductHandler) Handle() string {
	h.logger.Info("ProductHandler: handling request")
	return h.repo.FindProduct(123)
}

// HTTP Server
type Server struct {
	handlers []HTTPHandler
}

func NewServer(handlers []HTTPHandler) *Server {
	return &Server{handlers: handlers}
}

func (s *Server) Start() {
	fmt.Printf("Server starting with %d handlers\n", len(s.handlers))
	for i, h := range s.handlers {
		fmt.Printf("  Handler %d: %s\n", i, h.Handle())
	}
}
