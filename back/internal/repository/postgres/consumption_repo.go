package postgres

import (
	"context"
	"fmt"

	"github.com/RoggerParedesTavara/mi-recibo/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type ConsumptionRepository struct {
	pool *pgxpool.Pool
}

func NewConsumptionRepository(pool *pgxpool.Pool) *ConsumptionRepository {
	return &ConsumptionRepository{pool: pool}
}

// UpsertBatch saves or updates consumptions for a master bill
func (r *ConsumptionRepository) UpsertBatch(ctx context.Context, consumptions []domain.TenantConsumption) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error al iniciar transacción: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO tenant_consumptions (
			id, master_bill_id, tenant_id, consumed_units, previous_reading, current_reading,
			previous_month_consumption, variation_percentage, alert_spike, alert_message,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)
		ON CONFLICT (master_bill_id, tenant_id)
		DO UPDATE SET
			consumed_units = EXCLUDED.consumed_units,
			previous_reading = EXCLUDED.previous_reading,
			current_reading = EXCLUDED.current_reading,
			previous_month_consumption = EXCLUDED.previous_month_consumption,
			variation_percentage = EXCLUDED.variation_percentage,
			alert_spike = EXCLUDED.alert_spike,
			alert_message = EXCLUDED.alert_message,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id, created_at, updated_at
	`

	for i := range consumptions {
		c := &consumptions[i]
		if c.ID == uuid.Nil {
			c.ID = uuid.New()
		}
		err := tx.QueryRow(ctx, query,
			c.ID, c.MasterBillID, c.TenantID, c.ConsumedUnits, c.PreviousReading, c.CurrentReading,
			c.PreviousMonthConsumption, c.VariationPercentage, c.AlertSpike, c.AlertMessage,
		).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return fmt.Errorf("error al registrar consumo para inquilino %s: %w", c.TenantID, err)
		}
	}

	return tx.Commit(ctx)
}

func (r *ConsumptionRepository) ListByMasterBillID(ctx context.Context, masterBillID uuid.UUID) ([]domain.TenantConsumption, error) {
	query := `
		SELECT id, master_bill_id, tenant_id, consumed_units, previous_reading, current_reading,
		       previous_month_consumption, variation_percentage, alert_spike, alert_message,
		       created_at, updated_at
		FROM tenant_consumptions
		WHERE master_bill_id = $1
	`
	rows, err := r.pool.Query(ctx, query, masterBillID)
	if err != nil {
		return nil, fmt.Errorf("error al listar consumos: %w", err)
	}
	defer rows.Close()

	var consumptions []domain.TenantConsumption
	for rows.Next() {
		var c domain.TenantConsumption
		if err := rows.Scan(
			&c.ID, &c.MasterBillID, &c.TenantID, &c.ConsumedUnits, &c.PreviousReading, &c.CurrentReading,
			&c.PreviousMonthConsumption, &c.VariationPercentage, &c.AlertSpike, &c.AlertMessage,
			&c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("error al escanear consumo: %w", err)
		}
		consumptions = append(consumptions, c)
	}

	return consumptions, nil
}

// GetPreviousMonthConsumption finds the consumed units of the previous month for the same service and tenant
func (r *ConsumptionRepository) GetPreviousMonthConsumption(
	ctx context.Context,
	tenantID uuid.UUID,
	serviceType domain.ServiceType,
	year, month int,
) (*decimal.Decimal, error) {
	prevYear := year
	prevMonth := month - 1
	if prevMonth < 1 {
		prevMonth = 12
		prevYear--
	}

	query := `
		SELECT tc.consumed_units
		FROM tenant_consumptions tc
		JOIN master_bills mb ON mb.id = tc.master_bill_id
		WHERE tc.tenant_id = $1
		  AND mb.service_type = $2
		  AND mb.billing_year = $3
		  AND mb.billing_month = $4
		LIMIT 1
	`
	var units decimal.Decimal
	err := r.pool.QueryRow(ctx, query, tenantID, serviceType, prevYear, prevMonth).Scan(&units)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // No previous consumption recorded
		}
		return nil, fmt.Errorf("error al buscar consumo del mes anterior: %w", err)
	}

	return &units, nil
}
