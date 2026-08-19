package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/npbtrac/demo-go-user-management/internal/user"
)

func InternalMux(svc *user.Service) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("POST /internal/v1/users", createUser(svc))
	mux.HandleFunc("GET /internal/v1/users", listUsers(svc))
	mux.HandleFunc("GET /internal/v1/users/{id}", getUser(svc))
	mux.HandleFunc("PUT /internal/v1/users/{id}", putUser(svc))
	mux.HandleFunc("PATCH /internal/v1/users/{id}", patchUser(svc))
	mux.HandleFunc("DELETE /internal/v1/users/{id}", deleteUser(svc))
	return mux
}

func createUser(svc *user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body createJSON
		if err := decodeJSON(r, &body); err != nil {
			writeJSON(w, http.StatusBadRequest, errorJSON{Error: "invalid user payload: request body is invalid JSON"})
			return
		}
		u, err := svc.Create(r.Context(), user.CreateInput{
			Username: body.Username,
			Email:    body.Email,
			Phone:    body.Phone,
			Params:   body.Params,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, toJSON(u))
	}
}

func listUsers(svc *user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var pageSize int32
		if raw := r.URL.Query().Get("page_size"); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, errorJSON{Error: "invalid user payload: page_size is invalid"})
				return
			}
			pageSize = int32(n)
		}
		page, err := svc.List(r.Context(), pageSize, r.URL.Query().Get("page_token"))
		if err != nil {
			writeError(w, err)
			return
		}
		out := listJSON{Users: make([]userJSON, 0, len(page.Users)), NextPageToken: page.NextPageToken}
		for _, u := range page.Users {
			out.Users = append(out.Users, toJSON(u))
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func getUser(svc *user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := svc.Get(r.Context(), r.PathValue("id"))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toJSON(u))
	}
}

func putUser(svc *user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body createJSON
		if err := decodeJSON(r, &body); err != nil {
			writeJSON(w, http.StatusBadRequest, errorJSON{Error: "invalid user payload: request body is invalid JSON"})
			return
		}
		params := body.Params
		if len(params) == 0 {
			params = json.RawMessage(`{}`)
		}
		phone := ""
		if body.Phone != nil {
			phone = *body.Phone
		}
		u, err := svc.Update(r.Context(), r.PathValue("id"), user.UpdateInput{
			Username: &body.Username,
			Email:    &body.Email,
			Phone:    &phone,
			Params:   &params,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toJSON(u))
	}
}

func patchUser(svc *user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body updateJSON
		if err := decodeJSON(r, &body); err != nil {
			writeJSON(w, http.StatusBadRequest, errorJSON{Error: "invalid user payload: request body is invalid JSON"})
			return
		}
		in := user.UpdateInput{
			Username: body.Username,
			Email:    body.Email,
			Phone:    body.Phone,
		}
		if body.Params != nil {
			params := body.Params
			in.Params = &params
		}
		u, err := svc.Update(r.Context(), r.PathValue("id"), in)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toJSON(u))
	}
}

func deleteUser(svc *user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := svc.Delete(r.Context(), r.PathValue("id")); err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
