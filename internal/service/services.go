package service

import (
	"github.com/sharathm18/myapp/internal/repository"
	"github.com/sharathm18/myapp/internal/server"
)

type Services struct{}

func NewServices(s *server.APIServer, repos *repository.Repositories) (*Services, error) {
	return &Services{}, nil
}
