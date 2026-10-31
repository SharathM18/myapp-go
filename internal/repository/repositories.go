package repository

import "github.com/sharathm18/myapp/internal/server"

type Repositories struct {
}

func NewRepositories(s *server.APIServer) *Repositories {
	return &Repositories{}
}
