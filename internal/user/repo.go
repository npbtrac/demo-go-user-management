package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository stores users in Postgres.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository returns a Repository backed by pool.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, u User) (User, error) {
	const q = `
INSERT INTO users (id, username, email, phone, params, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, username, email, phone, params, created_at, updated_at`
	out, err := scanUser(r.pool.QueryRow(ctx, q, u.ID, u.Username, u.Email, u.Phone, u.Params, u.CreatedAt, u.UpdatedAt))
	return out, mapRepoErr(err)
}

func (r *PostgresRepository) Get(ctx context.Context, id string) (User, error) {
	const q = `
SELECT id, username, email, phone, params, created_at, updated_at
FROM users WHERE id = $1`
	out, err := scanUser(r.pool.QueryRow(ctx, q, id))
	return out, mapRepoErr(err)
}

func (r *PostgresRepository) List(ctx context.Context, limit, offset int) ([]User, error) {
	const q = `
SELECT id, username, email, phone, params, created_at, updated_at
FROM users
ORDER BY created_at DESC, id DESC
LIMIT $1 OFFSET $2`
	rows, err := r.pool.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, mapRepoErr(err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, mapRepoErr(err)
	}
	if users == nil {
		users = []User{}
	}
	return users, nil
}

func (r *PostgresRepository) Update(ctx context.Context, u User) (User, error) {
	const q = `
UPDATE users
SET username = $2, email = $3, phone = $4, params = $5, updated_at = $6
WHERE id = $1
RETURNING id, username, email, phone, params, created_at, updated_at`
	out, err := scanUser(r.pool.QueryRow(ctx, q, u.ID, u.Username, u.Email, u.Phone, u.Params, u.UpdatedAt))
	return out, mapRepoErr(err)
}

func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return mapRepoErr(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (User, error) {
	var u User
	var params []byte
	if err := row.Scan(&u.ID, &u.Username, &u.Email, &u.Phone, &params, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return User{}, err
	}
	u.Params = params
	u.CreatedAt = u.CreatedAt.UTC()
	u.UpdatedAt = u.UpdatedAt.UTC()
	return u, nil
}

func mapRepoErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		constraint := strings.ToLower(pgErr.ConstraintName)
		switch {
		case strings.Contains(constraint, "username"):
			return fmt.Errorf("%w: username already exists", ErrConflict)
		case strings.Contains(constraint, "email"):
			return fmt.Errorf("%w: email already exists", ErrConflict)
		default:
			return ErrConflict
		}
	}
	return err
}
