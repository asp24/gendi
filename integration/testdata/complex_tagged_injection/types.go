package main

type Middleware interface {
	Name() string
}

type AuthMiddleware struct{}
type LoggingMiddleware struct{}
type MetricsMiddleware struct{}

func NewAuthMiddleware() Middleware    { return &AuthMiddleware{} }
func NewLoggingMiddleware() Middleware { return &LoggingMiddleware{} }
func NewMetricsMiddleware() Middleware { return &MetricsMiddleware{} }

func (m *AuthMiddleware) Name() string    { return "auth" }
func (m *LoggingMiddleware) Name() string { return "logging" }
func (m *MetricsMiddleware) Name() string { return "metrics" }

type App struct {
	middleware []Middleware
}

func NewApp(middleware []Middleware) *App {
	return &App{middleware: middleware}
}
