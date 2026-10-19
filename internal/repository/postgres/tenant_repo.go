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

type TenantRepository struct {
	pool *pgxpool.Pool
}

func NewTenantRepository(pool *pgxpool.Pool) *TenantRepository {
	return &TenantRepository{pool: pool}
}

func (r *TenantRepository) Create(ctx context.Context, tenant *domain.Tenant) error {
	query := `
		INSERT INTO tenants (id, property_id, name, email, phone, room_or_unit, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id, created_at, updated_at
	`
	if tenant.ID == uuid.Nil {
		tenant.ID = uuid.New()
	}

	err := r.pool.QueryRow(ctx, query,
		tenant.ID, tenant.PropertyID, tenant.Name, tenant.Email, tenant.Phone, tenant.RoomOrUnit, tenant.IsActive).
		Scan(&tenant.ID, &tenant.CreatedAt, &tenant.UpdatedAt)
	if err != nil {
		return fmt.Errorf("error al insertar inquilino: %w", err)
	}

	return nil
}

func (r *TenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	query := `
		SELECT id, property_id, name, email, phone, room_or_unit, is_active, created_at, updated_at
		FROM tenants
		WHERE id = $1
	`
	tenant := &domain.Tenant{}
	err := r.pool.QueryRow(ctx, query, id).
		Scan(&tenant.ID, &tenant.PropertyID, &tenant.Name, &tenant.Email, &tenant.Phone,
			&tenant.RoomOrUnit, &tenant.IsActive, &tenant.CreatedAt, &tenant.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("error al obtener inquilino: %w", err)
	}

	return tenant, nil
}

func (r *TenantRepository) ListByPropertyID(ctx context.Context, propertyID uuid.UUID, onlyActive bool) ([]domain.Tenant, error) {
	query := `
		SELECT id, property_id, name, email, phone, room_or_unit, is_active, created_at, updated_at
		FROM tenants
		WHERE property_id = $1
	`
	if onlyActive {
		query += " AND is_active = true"
	}
	query += " ORDER BY name ASC"

	rows, err := r.pool.Query(ctx, query, propertyID)
	if err != nil {
		return nil, fmt.Errorf("error al listar inquilinos: %w", err)
	}
	defer rows.Close()

	var tenants []domain.Tenant
	for rows.Next() {
		var t domain.Tenant
		if err := rows.Scan(&t.ID, &t.PropertyID, &t.Name, &t.Email, &t.Phone,
			&t.RoomOrUnit, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("error al escanear inquilino: %w", err)
		}
		tenants = append(tenants, t)
	}

	return tenants, nil
}

func (r *TenantRepository) Update(ctx context.Context, tenant *domain.Tenant) error {
	query := `
		UPDATE tenants
		SET name = $2, email = $3, phone = $4, room_or_unit = $5, is_active = $6, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		tenant.ID, tenant.Name, tenant.Email, tenant.Phone, tenant.RoomOrUnit, tenant.IsActive).
		Scan(&tenant.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("error al actualizar inquilino: %w", err)
	}

	return nil
}
