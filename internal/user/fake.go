package user

import (
	"context"
	"sort"
	"strings"
	"sync"
)

// FakeRepository is an in-memory Repository for unit tests.
type FakeRepository struct {
	mu    sync.Mutex
	users map[string]User
}

// NewFakeRepository returns an empty in-memory repository.
func NewFakeRepository() *FakeRepository {
	return &FakeRepository{users: make(map[string]User)}
}

func (r *FakeRepository) Create(_ context.Context, u User) (User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.users {
		if strings.EqualFold(existing.Username, u.Username) {
			return User{}, fmtConflict("username")
		}
		if strings.EqualFold(existing.Email, u.Email) {
			return User{}, fmtConflict("email")
		}
	}
	cp := u
	r.users[u.ID] = cp
	return cp, nil
}

func (r *FakeRepository) Get(_ context.Context, id string) (User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

func (r *FakeRepository) List(_ context.Context, limit, offset int) ([]User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	all := make([]User, 0, len(r.users))
	for _, u := range r.users {
		all = append(all, u)
	}
	sort.Slice(all, func(i, j int) bool {
		if !all[i].CreatedAt.Equal(all[j].CreatedAt) {
			return all[i].CreatedAt.After(all[j].CreatedAt)
		}
		return all[i].ID > all[j].ID
	})
	if offset > len(all) {
		return []User{}, nil
	}
	all = all[offset:]
	if limit < len(all) {
		all = all[:limit]
	}
	return all, nil
}

func (r *FakeRepository) Update(_ context.Context, u User) (User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.users[u.ID]; !ok {
		return User{}, ErrNotFound
	}
	for id, existing := range r.users {
		if id == u.ID {
			continue
		}
		if strings.EqualFold(existing.Username, u.Username) {
			return User{}, fmtConflict("username")
		}
		if strings.EqualFold(existing.Email, u.Email) {
			return User{}, fmtConflict("email")
		}
	}
	r.users[u.ID] = u
	return u, nil
}

func (r *FakeRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.users[id]; !ok {
		return ErrNotFound
	}
	delete(r.users, id)
	return nil
}

func fmtConflict(field string) error {
	return wrapConflict(field)
}
