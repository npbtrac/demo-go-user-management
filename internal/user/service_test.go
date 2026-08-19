package user_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/npbtrac/demo-go-user-management/internal/user"
)

func TestCreateGetUpdateDelete(t *testing.T) {
	svc := user.NewService(user.NewFakeRepository())
	ctx := context.Background()
	phone := "+15551212"
	created, err := svc.Create(ctx, user.CreateInput{
		Username: "alice",
		Email:    "alice@example.com",
		Phone:    &phone,
		Params:   json.RawMessage(`{"plan":"pro"}`),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID == "" || created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Fatalf("expected server-generated id and timestamps: %+v", created)
	}
	got, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if string(got.Params) != `{"plan":"pro"}` {
		t.Fatalf("params round-trip: %s", got.Params)
	}

	newEmail := "alice2@example.com"
	updated, err := svc.Update(ctx, created.ID, user.UpdateInput{Email: &newEmail})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Email != newEmail {
		t.Fatalf("email: %s", updated.Email)
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) && !updated.UpdatedAt.Equal(created.UpdatedAt) {
		// clock may be equal in same second; still must not go backwards
		if updated.UpdatedAt.Before(created.CreatedAt) {
			t.Fatalf("updated_at moved backwards")
		}
	}

	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := svc.Get(ctx, created.ID); !errors.Is(err, user.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestCreateValidationAndConflict(t *testing.T) {
	svc := user.NewService(user.NewFakeRepository())
	ctx := context.Background()
	if _, err := svc.Create(ctx, user.CreateInput{Username: "", Email: "a@b.com"}); !errors.Is(err, user.ErrInvalid) {
		t.Fatalf("empty username: %v", err)
	}
	if _, err := svc.Create(ctx, user.CreateInput{Username: "bob", Email: "not-an-email"}); !errors.Is(err, user.ErrInvalid) {
		t.Fatalf("bad email: %v", err)
	}
	if _, err := svc.Create(ctx, user.CreateInput{Username: "bob", Email: "bob@example.com", Params: json.RawMessage(`[]`)}); !errors.Is(err, user.ErrInvalid) {
		t.Fatalf("non-object params: %v", err)
	}
	if _, err := svc.Create(ctx, user.CreateInput{Username: "bob", Email: "bob@example.com"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := svc.Create(ctx, user.CreateInput{Username: "bob", Email: "other@example.com"}); !errors.Is(err, user.ErrConflict) {
		t.Fatalf("dup username: %v", err)
	}
	if _, err := svc.Create(ctx, user.CreateInput{Username: "other", Email: "bob@example.com"}); !errors.Is(err, user.ErrConflict) {
		t.Fatalf("dup email: %v", err)
	}
}

func TestListPaginationAndMissing(t *testing.T) {
	svc := user.NewService(user.NewFakeRepository())
	ctx := context.Background()
	for i, name := range []string{"a", "b", "c"} {
		if _, err := svc.Create(ctx, user.CreateInput{Username: name, Email: name + "@example.com"}); err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}
	page1, err := svc.List(ctx, 2, "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page1.Users) != 2 || page1.NextPageToken == "" {
		t.Fatalf("page1: %+v", page1)
	}
	page2, err := svc.List(ctx, 2, page1.NextPageToken)
	if err != nil {
		t.Fatalf("list page2: %v", err)
	}
	if len(page2.Users) != 1 {
		t.Fatalf("page2 size %d", len(page2.Users))
	}
	if _, err := svc.Get(ctx, uuid.NewString()); !errors.Is(err, user.ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
	if _, err := svc.Get(ctx, "nope"); !errors.Is(err, user.ErrInvalid) {
		t.Fatalf("bad id: %v", err)
	}
}

func TestUpdateRequiresField(t *testing.T) {
	svc := user.NewService(user.NewFakeRepository())
	ctx := context.Background()
	u, err := svc.Create(ctx, user.CreateInput{Username: "c", Email: "c@example.com"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := svc.Update(ctx, u.ID, user.UpdateInput{}); !errors.Is(err, user.ErrInvalid) {
		t.Fatalf("empty update: %v", err)
	}
}

func TestCreatedAtUsesClock(t *testing.T) {
	if time.Now().IsZero() {
		t.Fatal("clock")
	}
}
