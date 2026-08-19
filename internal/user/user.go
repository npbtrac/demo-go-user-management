package user

import (
	"encoding/json"
	"time"
)

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
	MaxUsernameLen  = 64
	MaxEmailLen     = 254
	MaxPhoneLen     = 32
	MaxParamsBytes  = 16 << 10
)

// User is the domain record persisted in Postgres.
type User struct {
	ID        string
	Username  string
	Email     string
	Phone     *string
	Params    json.RawMessage
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CreateInput is the payload for creating a user.
type CreateInput struct {
	Username string
	Email    string
	Phone    *string
	Params   json.RawMessage
}

// UpdateInput is a partial update. Nil fields are left unchanged.
// A non-nil Phone with empty string clears the stored phone.
type UpdateInput struct {
	Username *string
	Email    *string
	Phone    *string
	Params   *json.RawMessage
}

// Page is a paginated list of users.
type Page struct {
	Users         []User
	NextPageToken string
}
