package main

type Service struct {
	banner string
}

func NewService(banner string) *Service {
	return &Service{banner: banner}
}

func (s *Service) GetBanner() string {
	return s.banner
}
