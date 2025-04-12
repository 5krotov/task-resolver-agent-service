package hello_world_service

import (
	"fmt"
	"net/http"
)

type HelloWorldHandler struct {
}

func NewHelloWorldHandler() *HelloWorldHandler {
	return &HelloWorldHandler{}
}

func (h *HelloWorldHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	fmt.Fprintln(w, "Hello, World!")
}
