package hello_world_service

import "net/http"

type HelloWorldService struct {
	handler *HelloWorldHandler
}

func NewHelloWorldService() *HelloWorldService {
	return &HelloWorldService{handler: NewHelloWorldHandler()}
}

func (s *HelloWorldService) Register(mux *http.ServeMux) {
	mux.HandleFunc("/", s.handler.Handle)
}
