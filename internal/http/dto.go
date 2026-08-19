package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/npbtrac/demo-go-user-management/internal/user"
)

type userJSON struct {
	ID        string          `json:"id"`
	Username  string          `json:"username"`
	Email     string          `json:"email"`
	Phone     *string         `json:"phone"`
	Params    json.RawMessage `json:"params"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type createJSON struct {
	Username string          `json:"username"`
	Email    string          `json:"email"`
	Phone    *string         `json:"phone"`
	Params   json.RawMessage `json:"params"`
}

type updateJSON struct {
	Username *string         `json:"username"`
	Email    *string         `json:"email"`
	Phone    *string         `json:"phone"`
	Params   json.RawMessage `json:"params"`
}

type listJSON struct {
	Users         []userJSON `json:"users"`
	NextPageToken string     `json:"next_page_token,omitempty"`
}

type errorJSON struct {
	Error string `json:"error"`
}

func toJSON(u user.User) userJSON {
	params := u.Params
	if len(params) == 0 {
		params = json.RawMessage(`{}`)
	}
	return userJSON{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Phone:     u.Phone,
		Params:    params,
		CreatedAt: u.CreatedAt.UTC(),
		UpdatedAt: u.UpdatedAt.UTC(),
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, user.ErrInvalid):
		writeJSON(w, http.StatusBadRequest, errorJSON{Error: err.Error()})
	case errors.Is(err, user.ErrNotFound):
		writeJSON(w, http.StatusNotFound, errorJSON{Error: err.Error()})
	case errors.Is(err, user.ErrConflict):
		writeJSON(w, http.StatusConflict, errorJSON{Error: err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, errorJSON{Error: "internal error"})
	}
}

func decodeJSON(r *http.Request, dest any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dest); err != nil {
		return err
	}
	return nil
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
