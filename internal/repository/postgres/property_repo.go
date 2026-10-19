package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/RoggerParedesTavara/mi-recibo/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PropertyRepository struct {
	pool *pgxpool.Pool
}

func NewPropertyRepository(pool *pgxpool.Pool) *PropertyRepository {
	return &PropertyRepository{pool: pool}
}

func (r *PropertyRepository) Create(ctx context.Context, property *domain.Property) error {
	query := `
		INSERT INTO properties (id, owner_id, name, address, created_at, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id, created_at, updated_at
	`
	if property.ID == uuid.Nil {
		property.ID = uuid.New()
	}

	err := r.pool.QueryRow(ctx, query, property.ID, property.OwnerID, property.Name, property.Address).
		Scan(&property.ID, &property.CreatedAt, &property.UpdatedAt)
	if err != nil {
		return fmt.Errorf("error al insertar propiedad: %w", err)
	}

	return nil
}

func (r *PropertyRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Property, error) {
	query := `
		SELECT id, owner_id, name, address, created_at, updated_at
		FROM properties
		WHERE id = $1
	`
	prop := &domain.Property{}
	err := r.pool.QueryRow(ctx, query, id).
		Scan(&prop.ID, &prop.OwnerID, &prop.Name, &prop.Address, &prop.CreatedAt, &prop.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("error al obtener propiedad: %w", err)
	}

	return prop, nil
}

func (r *PropertyRepository) ListByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]domain.Property, error) {
	query := `
		SELECT id, owner_id, name, address, created_at, updated_at
		FROM properties
		WHERE owner_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, ownerID)
	if err != nil {
		return nil, fmt.Errorf("error al listar propiedades: %w", err)
	}
	defer rows.Close()

	var properties []domain.Property
	for rows.Next() {
		var p domain.Property
		if err := rows.Scan(&p.ID, &p.OwnerID, &p.Name, &p.Address, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("error al escanear propiedad: %w", err)
		}
		properties = append(properties, p)
	}

	return properties, nil
}
