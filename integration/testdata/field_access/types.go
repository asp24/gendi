package main

import "fmt"

type DatabaseConfig struct {
	DSN string
}

type AppConfig struct {
	Host     string
	Port     int
	Database DatabaseConfig
}

func LoadConfig() *AppConfig {
	return &AppConfig{
		Host:     "localhost",
		Port:     8080,
		Database: DatabaseConfig{DSN: "postgres://localhost/mydb"},
	}
}

type Server struct {
	host string
	dsn  string
}

func NewServer(host string, dsn string) *Server {
	return &Server{host: host, dsn: dsn}
}

func (s *Server) Info() string {
	return fmt.Sprintf("host=%s dsn=%s", s.host, s.dsn)
}
