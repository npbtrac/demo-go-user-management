package httpserver

import (
	"net/http"

	"github.com/npbtrac/demo-go-user-management/internal/user"
)

func PublicMux(svc *user.Service) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("GET /public/v1/users/{id}", getUser(svc))
	return mux
}
