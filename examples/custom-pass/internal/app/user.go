package app

import (
	"fmt"
	"log/slog"
)

type UserRepository interface {
	FindUser(id int) string
}

type UserRepoImpl struct {
	dsn string
}

func NewUserRepository(dsn string) *UserRepoImpl {
	return &UserRepoImpl{dsn: dsn}
}

func (r *UserRepoImpl) FindUser(id int) string {
	return fmt.Sprintf("User #%d from %s", id, r.dsn)
}

type UserHandler struct {
	logger *slog.Logger
	repo   UserRepository
}

func NewUserHandler(logger *slog.Logger, repo UserRepository) HTTPHandler {
	return &UserHandler{logger: logger, repo: repo}
}

func (h *UserHandler) Handle() string {
	h.logger.Info("UserHandler: handling request")
	return h.repo.FindUser(42)
}
