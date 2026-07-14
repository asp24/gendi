package main

import "fmt"

type DatabaseConfig struct {
	DSN string
}

type AppConfig struct {
	Host     string
	Database DatabaseConfig
}

var Defaults = AppConfig{
	Host:     "localhost",
	Database: DatabaseConfig{DSN: "postgres://localhost/defaults"},
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
