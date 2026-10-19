package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/RoggerParedesTavara/mi-recibo/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TenantBillRepository struct {
	pool *pgxpool.Pool
}

func NewTenantBillRepository(pool *pgxpool.Pool) *TenantBillRepository {
	return &TenantBillRepository{pool: pool}
}

// SaveCalculationResult persists generated tenant bills and their itemized extra charge breakdowns atomically
func (r *TenantBillRepository) SaveCalculationResult(ctx context.Context, masterBillID uuid.UUID, bills []domain.TenantBill) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error al iniciar transacción: %w", err)
	}
	defer tx.Rollback(ctx)

	// Clean up existing bills for this master_bill if recalculating
	_, err = tx.Exec(ctx, `DELETE FROM tenant_bills WHERE master_bill_id = $1`, masterBillID)
	if err != nil {
		return fmt.Errorf("error al limpiar recibos anteriores: %w", err)
	}

	insertBillQuery := `
		INSERT INTO tenant_bills (
			id, master_bill_id, tenant_id, consumed_units, unit_price,
			consumption_amount, extra_charges_total_share, total_amount, status,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)
		RETURNING created_at, updated_at
	`

	insertBreakdownQuery := `
		INSERT INTO tenant_bill_charge_breakdowns (
			id, tenant_bill_id, extra_charge_id, description, original_amount,
			tenants_count, tenant_share, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP
		)
		RETURNING created_at
	`

	for i := range bills {
		b := &bills[i]
		if b.ID == uuid.Nil {
			b.ID = uuid.New()
		}

		err = tx.QueryRow(ctx, insertBillQuery,
			b.ID, b.MasterBillID, b.TenantID, b.ConsumedUnits, b.UnitPrice,
			b.ConsumptionAmount, b.ExtraChargesTotalShare, b.TotalAmount, b.Status,
		).Scan(&b.CreatedAt, &b.UpdatedAt)
		if err != nil {
			return fmt.Errorf("error al guardar recibo de inquilino: %w", err)
		}

		// Insert itemized breakdowns
		for j := range b.Breakdowns {
			bd := &b.Breakdowns[j]
			if bd.ID == uuid.Nil {
				bd.ID = uuid.New()
			}
			bd.TenantBillID = b.ID

			err = tx.QueryRow(ctx, insertBreakdownQuery,
				bd.ID, bd.TenantBillID, bd.ExtraChargeID, bd.Description, bd.OriginalAmount,
				bd.TenantsCount, bd.TenantShare,
			).Scan(&bd.CreatedAt)
			if err != nil {
				return fmt.Errorf("error al guardar desglose de cargo '%s': %w", bd.Description, err)
			}
		}
	}

	// Update master bill status to CALCULATED
	_, err = tx.Exec(ctx, `
		UPDATE master_bills
		SET status = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, masterBillID, domain.BillStatusCalculated)
	if err != nil {
		return fmt.Errorf("error al actualizar estado de recibo principal: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *TenantBillRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.TenantBillDetail, error) {
	query := `
		SELECT tb.id, tb.master_bill_id, tb.tenant_id, tb.consumed_units, tb.unit_price,
		       tb.consumption_amount, tb.extra_charges_total_share, tb.total_amount, tb.status,
		       tb.paid_at, tb.created_at, tb.updated_at,
		       t.name AS tenant_name, t.room_or_unit, p.name AS property_name,
		       mb.service_type, mb.billing_year, mb.billing_month,
		       COALESCE(tc.alert_spike, false) AS alert_spike,
		       COALESCE(tc.alert_message, '') AS alert_message,
		       tc.previous_month_consumption
		FROM tenant_bills tb
		JOIN tenants t ON t.id = tb.tenant_id
		JOIN master_bills mb ON mb.id = tb.master_bill_id
		JOIN properties p ON p.id = mb.property_id
		LEFT JOIN tenant_consumptions tc ON tc.master_bill_id = tb.master_bill_id AND tc.tenant_id = tb.tenant_id
		WHERE tb.id = $1
	`
	detail := &domain.TenantBillDetail{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&detail.ID, &detail.MasterBillID, &detail.TenantID, &detail.ConsumedUnits, &detail.UnitPrice,
		&detail.ConsumptionAmount, &detail.ExtraChargesTotalShare, &detail.TotalAmount, &detail.Status,
		&detail.PaidAt, &detail.CreatedAt, &detail.UpdatedAt,
		&detail.TenantName, &detail.RoomOrUnit, &detail.PropertyName,
		&detail.ServiceType, &detail.BillingYear, &detail.BillingMonth,
		&detail.AlertSpike, &detail.AlertMessage, &detail.PreviousMonthUnits,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("error al obtener recibo de inquilino: %w", err)
	}

	// Fetch breakdowns
	breakdowns, err := r.getBreakdowns(ctx, detail.ID)
	if err != nil {
		return nil, err
	}
	detail.Breakdowns = breakdowns

	return detail, nil
}

func (r *TenantBillRepository) ListByMasterBillID(ctx context.Context, masterBillID uuid.UUID) ([]domain.TenantBillDetail, error) {
	query := `
		SELECT tb.id, tb.master_bill_id, tb.tenant_id, tb.consumed_units, tb.unit_price,
		       tb.consumption_amount, tb.extra_charges_total_share, tb.total_amount, tb.status,
		       tb.paid_at, tb.created_at, tb.updated_at,
		       t.name AS tenant_name, t.room_or_unit, p.name AS property_name,
		       mb.service_type, mb.billing_year, mb.billing_month,
		       COALESCE(tc.alert_spike, false) AS alert_spike,
		       COALESCE(tc.alert_message, '') AS alert_message,
		       tc.previous_month_consumption
		FROM tenant_bills tb
		JOIN tenants t ON t.id = tb.tenant_id
		JOIN master_bills mb ON mb.id = tb.master_bill_id
		JOIN properties p ON p.id = mb.property_id
		LEFT JOIN tenant_consumptions tc ON tc.master_bill_id = tb.master_bill_id AND tc.tenant_id = tb.tenant_id
		WHERE tb.master_bill_id = $1
		ORDER BY t.name ASC
	`
	rows, err := r.pool.Query(ctx, query, masterBillID)
	if err != nil {
		return nil, fmt.Errorf("error al listar recibos de inquilinos: %w", err)
	}
	defer rows.Close()

	var details []domain.TenantBillDetail
	for rows.Next() {
		var d domain.TenantBillDetail
		if err := rows.Scan(
			&d.ID, &d.MasterBillID, &d.TenantID, &d.ConsumedUnits, &d.UnitPrice,
			&d.ConsumptionAmount, &d.ExtraChargesTotalShare, &d.TotalAmount, &d.Status,
			&d.PaidAt, &d.CreatedAt, &d.UpdatedAt,
			&d.TenantName, &d.RoomOrUnit, &d.PropertyName,
			&d.ServiceType, &d.BillingYear, &d.BillingMonth,
			&d.AlertSpike, &d.AlertMessage, &d.PreviousMonthUnits,
		); err != nil {
			return nil, fmt.Errorf("error al escanear recibo de inquilino: %w", err)
		}
		details = append(details, d)
	}

	// Fetch breakdowns for each bill
	for i := range details {
		bds, err := r.getBreakdowns(ctx, details[i].ID)
		if err != nil {
			return nil, err
		}
		details[i].Breakdowns = bds
	}

	return details, nil
}

func (r *TenantBillRepository) ListByTenantID(ctx context.Context, tenantID uuid.UUID) ([]domain.TenantBillDetail, error) {
	query := `
		SELECT tb.id, tb.master_bill_id, tb.tenant_id, tb.consumed_units, tb.unit_price,
		       tb.consumption_amount, tb.extra_charges_total_share, tb.total_amount, tb.status,
		       tb.paid_at, tb.created_at, tb.updated_at,
		       t.name AS tenant_name, t.room_or_unit, p.name AS property_name,
		       mb.service_type, mb.billing_year, mb.billing_month,
		       COALESCE(tc.alert_spike, false) AS alert_spike,
		       COALESCE(tc.alert_message, '') AS alert_message,
		       tc.previous_month_consumption
		FROM tenant_bills tb
		JOIN tenants t ON t.id = tb.tenant_id
		JOIN master_bills mb ON mb.id = tb.master_bill_id
		JOIN properties p ON p.id = mb.property_id
		LEFT JOIN tenant_consumptions tc ON tc.master_bill_id = tb.master_bill_id AND tc.tenant_id = tb.tenant_id
		WHERE tb.tenant_id = $1
		ORDER BY mb.billing_year DESC, mb.billing_month DESC
	`
	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("error al listar recibos por inquilino: %w", err)
	}
	defer rows.Close()

	var details []domain.TenantBillDetail
	for rows.Next() {
		var d domain.TenantBillDetail
		if err := rows.Scan(
			&d.ID, &d.MasterBillID, &d.TenantID, &d.ConsumedUnits, &d.UnitPrice,
			&d.ConsumptionAmount, &d.ExtraChargesTotalShare, &d.TotalAmount, &d.Status,
			&d.PaidAt, &d.CreatedAt, &d.UpdatedAt,
			&d.TenantName, &d.RoomOrUnit, &d.PropertyName,
			&d.ServiceType, &d.BillingYear, &d.BillingMonth,
			&d.AlertSpike, &d.AlertMessage, &d.PreviousMonthUnits,
		); err != nil {
			return nil, fmt.Errorf("error al escanear recibo: %w", err)
		}
		details = append(details, d)
	}

	for i := range details {
		bds, err := r.getBreakdowns(ctx, details[i].ID)
		if err != nil {
			return nil, err
		}
		details[i].Breakdowns = bds
	}

	return details, nil
}

func (r *TenantBillRepository) UpdatePaymentStatus(ctx context.Context, id uuid.UUID, status domain.TenantBillStatus) error {
	var paidAt *time.Time
	if status == domain.TenantBillPaid {
		now := time.Now()
		paidAt = &now
	}

	query := `
		UPDATE tenant_bills
		SET status = $2, paid_at = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	cmdTag, err := r.pool.Exec(ctx, query, id, status, paidAt)
	if err != nil {
		return fmt.Errorf("error al actualizar estado de pago: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *TenantBillRepository) getBreakdowns(ctx context.Context, tenantBillID uuid.UUID) ([]domain.TenantBillChargeBreakdown, error) {
	query := `
		SELECT id, tenant_bill_id, extra_charge_id, description, original_amount,
		       tenants_count, tenant_share, created_at
		FROM tenant_bill_charge_breakdowns
		WHERE tenant_bill_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, tenantBillID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener desgloses: %w", err)
	}
	defer rows.Close()

	var list []domain.TenantBillChargeBreakdown
	for rows.Next() {
		var b domain.TenantBillChargeBreakdown
		if err := rows.Scan(
			&b.ID, &b.TenantBillID, &b.ExtraChargeID, &b.Description, &b.OriginalAmount,
			&b.TenantsCount, &b.TenantShare, &b.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("error al escanear desglose: %w", err)
		}
		list = append(list, b)
	}
	return list, nil
}
