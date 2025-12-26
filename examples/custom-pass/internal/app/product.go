package app

import (
	"fmt"
	"log/slog"
)

type ProductRepository interface {
	FindProduct(id int) string
}

type ProductRepoImpl struct {
	dsn string
}

func NewProductRepository(dsn string) *ProductRepoImpl {
	return &ProductRepoImpl{dsn: dsn}
}

func (r *ProductRepoImpl) FindProduct(id int) string {
	return fmt.Sprintf("Product #%d from %s", id, r.dsn)
}

type ProductHandler struct {
	logger *slog.Logger
	repo   ProductRepository
}

func NewProductHandler(logger *slog.Logger, repo ProductRepository) HTTPHandler {
	return &ProductHandler{logger: logger, repo: repo}
}

func (h *ProductHandler) Handle() string {
	h.logger.Info("ProductHandler: handling request")

	return h.repo.FindProduct(123)
}
