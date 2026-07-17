//go:build taggedsvc

package tagged

// Service is only compiled when the "taggedsvc" build tag is set.
type Service struct{}

func NewService() *Service {
	return &Service{}
}
