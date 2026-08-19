package user

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Repository persists users.
type Repository interface {
	Create(ctx context.Context, u User) (User, error)
	Get(ctx context.Context, id string) (User, error)
	List(ctx context.Context, limit, offset int) ([]User, error)
	Update(ctx context.Context, u User) (User, error)
	Delete(ctx context.Context, id string) error
}

// Service is the application API used by HTTP and gRPC adapters.
type Service struct {
	repo Repository
	now  func() time.Time
	id   func() string
}

// NewService returns a Service backed by repo.
func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
		now:  func() time.Time { return time.Now().UTC() },
		id:   func() string { return uuid.NewString() },
	}
}

// Create inserts a user and returns the stored record.
func (s *Service) Create(ctx context.Context, in CreateInput) (User, error) {
	if err := validateCreate(in); err != nil {
		return User{}, err
	}
	params, err := normalizeParams(in.Params)
	if err != nil {
		return User{}, err
	}
	now := s.now()
	u := User{
		ID:        s.id(),
		Username:  strings.TrimSpace(in.Username),
		Email:     strings.TrimSpace(in.Email),
		Phone:     normalizePhone(in.Phone),
		Params:    params,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return s.repo.Create(ctx, u)
}

// Get returns a user by id.
func (s *Service) Get(ctx context.Context, id string) (User, error) {
	if err := validateID(id); err != nil {
		return User{}, err
	}
	return s.repo.Get(ctx, id)
}

// List returns a page of users.
func (s *Service) List(ctx context.Context, pageSize int32, pageToken string) (Page, error) {
	limit := int(pageSize)
	if limit <= 0 {
		limit = DefaultPageSize
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}
	offset, err := decodePageToken(pageToken)
	if err != nil {
		return Page{}, err
	}
	users, err := s.repo.List(ctx, limit+1, offset)
	if err != nil {
		return Page{}, err
	}
	page := Page{Users: users}
	if len(users) > limit {
		page.Users = users[:limit]
		page.NextPageToken = encodePageToken(offset + limit)
	}
	return page, nil
}

// Update applies a partial update.
func (s *Service) Update(ctx context.Context, id string, in UpdateInput) (User, error) {
	if err := validateID(id); err != nil {
		return User{}, err
	}
	if in.Username == nil && in.Email == nil && in.Phone == nil && in.Params == nil {
		return User{}, fmt.Errorf("%w: at least one field is required", ErrInvalid)
	}
	current, err := s.repo.Get(ctx, id)
	if err != nil {
		return User{}, err
	}
	if in.Username != nil {
		if err := validateUsername(*in.Username); err != nil {
			return User{}, err
		}
		current.Username = strings.TrimSpace(*in.Username)
	}
	if in.Email != nil {
		if err := validateEmail(*in.Email); err != nil {
			return User{}, err
		}
		current.Email = strings.TrimSpace(*in.Email)
	}
	if in.Phone != nil {
		if err := validatePhone(in.Phone); err != nil {
			return User{}, err
		}
		current.Phone = normalizePhone(in.Phone)
	}
	if in.Params != nil {
		params, err := normalizeParams(*in.Params)
		if err != nil {
			return User{}, err
		}
		current.Params = params
	}
	current.UpdatedAt = s.now()
	return s.repo.Update(ctx, current)
}

// Delete removes a user by id.
func (s *Service) Delete(ctx context.Context, id string) error {
	if err := validateID(id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

func validateID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("%w: id is required", ErrInvalid)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: id must be a UUID", ErrInvalid)
	}
	return nil
}

func encodePageToken(offset int) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

func decodePageToken(token string) (int, error) {
	if strings.TrimSpace(token) == "" {
		return 0, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return 0, fmt.Errorf("%w: page_token is invalid", ErrInvalid)
	}
	offset, err := strconv.Atoi(string(b))
	if err != nil || offset < 0 {
		return 0, fmt.Errorf("%w: page_token is invalid", ErrInvalid)
	}
	return offset, nil
}
