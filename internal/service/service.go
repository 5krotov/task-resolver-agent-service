package service

import "net/http"

type Service interface {
	Register(mux *http.ServeMux)
}
