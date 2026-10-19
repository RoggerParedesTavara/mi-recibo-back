package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/RoggerParedesTavara/mi-recibo/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OwnerRepository struct {
	pool *pgxpool.Pool
}

func NewOwnerRepository(pool *pgxpool.Pool) *OwnerRepository {
	return &OwnerRepository{pool: pool}
}

func (r *OwnerRepository) Create(ctx context.Context, owner *domain.Owner) error {
	query := `
		INSERT INTO owners (id, name, email, phone, created_at, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id, created_at, updated_at
	`
	if owner.ID == uuid.Nil {
		owner.ID = uuid.New()
	}

	err := r.pool.QueryRow(ctx, query, owner.ID, owner.Name, owner.Email, owner.Phone).
		Scan(&owner.ID, &owner.CreatedAt, &owner.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique violation
			return domain.ErrConflict
		}
		return fmt.Errorf("error al insertar propietario: %w", err)
	}

	return nil
}

func (r *OwnerRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Owner, error) {
	query := `
		SELECT id, name, email, phone, created_at, updated_at
		FROM owners
		WHERE id = $1
	`
	owner := &domain.Owner{}
	err := r.pool.QueryRow(ctx, query, id).
		Scan(&owner.ID, &owner.Name, &owner.Email, &owner.Phone, &owner.CreatedAt, &owner.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("error al obtener propietario: %w", err)
	}

	return owner, nil
}
