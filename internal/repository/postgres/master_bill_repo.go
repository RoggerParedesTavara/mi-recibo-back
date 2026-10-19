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
	"github.com/shopspring/decimal"
)

type MasterBillRepository struct {
	pool *pgxpool.Pool
}

func NewMasterBillRepository(pool *pgxpool.Pool) *MasterBillRepository {
	return &MasterBillRepository{pool: pool}
}

func (r *MasterBillRepository) Create(ctx context.Context, bill *domain.MasterBill) error {
	if bill.ID == uuid.Nil {
		bill.ID = uuid.New()
	}

	// Calculate totals
	bill.ConsumptionAmount = bill.TotalConsumption.Mul(bill.UnitPrice).Round(2)

	extraTotal := decimal.Zero
	for _, extra := range bill.ExtraCharges {
		extraTotal = extraTotal.Add(extra.Amount)
	}
	bill.ExtraChargesTotal = extraTotal.Round(2)
	bill.TotalAmount = bill.ConsumptionAmount.Add(bill.ExtraChargesTotal).Round(2)

	if bill.Status == "" {
		bill.Status = domain.BillStatusDraft
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error al iniciar transacción: %w", err)
	}
	defer tx.Rollback(ctx)

	insertBillQuery := `
		INSERT INTO master_bills (
			id, property_id, service_type, billing_year, billing_month,
			total_consumption, unit_price, consumption_amount, extra_charges_total,
			total_amount, issue_date, due_date, status, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)
		RETURNING created_at, updated_at
	`
	err = tx.QueryRow(ctx, insertBillQuery,
		bill.ID, bill.PropertyID, bill.ServiceType, bill.BillingYear, bill.BillingMonth,
		bill.TotalConsumption, bill.UnitPrice, bill.ConsumptionAmount, bill.ExtraChargesTotal,
		bill.TotalAmount, bill.IssueDate, bill.DueDate, bill.Status,
	).Scan(&bill.CreatedAt, &bill.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique violation on period
			return domain.ErrConflict
		}
		return fmt.Errorf("error al insertar recibo principal: %w", err)
	}

	// Insert dynamic list of extra charges
	insertExtraQuery := `
		INSERT INTO master_bill_extra_charges (id, master_bill_id, description, amount, created_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
		RETURNING created_at
	`
	for i := range bill.ExtraCharges {
		charge := &bill.ExtraCharges[i]
		if charge.ID == uuid.Nil {
			charge.ID = uuid.New()
		}
		charge.MasterBillID = bill.ID
		charge.Amount = charge.Amount.Round(2)

		err = tx.QueryRow(ctx, insertExtraQuery, charge.ID, charge.MasterBillID, charge.Description, charge.Amount).
			Scan(&charge.CreatedAt)
		if err != nil {
			return fmt.Errorf("error al insertar cargo extra '%s': %w", charge.Description, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error al commitear recibo principal: %w", err)
	}

	return nil
}

func (r *MasterBillRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.MasterBill, error) {
	billQuery := `
		SELECT id, property_id, service_type, billing_year, billing_month,
		       total_consumption, unit_price, consumption_amount, extra_charges_total,
		       total_amount, issue_date, due_date, status, created_at, updated_at
		FROM master_bills
		WHERE id = $1
	`
	bill := &domain.MasterBill{}
	err := r.pool.QueryRow(ctx, billQuery, id).Scan(
		&bill.ID, &bill.PropertyID, &bill.ServiceType, &bill.BillingYear, &bill.BillingMonth,
		&bill.TotalConsumption, &bill.UnitPrice, &bill.ConsumptionAmount, &bill.ExtraChargesTotal,
		&bill.TotalAmount, &bill.IssueDate, &bill.DueDate, &bill.Status, &bill.CreatedAt, &bill.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("error al obtener recibo principal: %w", err)
	}

	// Fetch dynamic extra charges
	chargesQuery := `
		SELECT id, master_bill_id, description, amount, created_at
		FROM master_bill_extra_charges
		WHERE master_bill_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, chargesQuery, id)
	if err != nil {
		return nil, fmt.Errorf("error al obtener cargos extras: %w", err)
	}
	defer rows.Close()

	bill.ExtraCharges = make([]domain.MasterBillExtraCharge, 0)
	for rows.Next() {
		var c domain.MasterBillExtraCharge
		if err := rows.Scan(&c.ID, &c.MasterBillID, &c.Description, &c.Amount, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("error al escanear cargo extra: %w", err)
		}
		bill.ExtraCharges = append(bill.ExtraCharges, c)
	}

	return bill, nil
}

func (r *MasterBillRepository) ListByProperty(ctx context.Context, propertyID uuid.UUID, year, month int, service domain.ServiceType) ([]domain.MasterBill, error) {
	query := `
		SELECT id, property_id, service_type, billing_year, billing_month,
		       total_consumption, unit_price, consumption_amount, extra_charges_total,
		       total_amount, issue_date, due_date, status, created_at, updated_at
		FROM master_bills
		WHERE property_id = $1
	`
	args := []any{propertyID}
	argIdx := 2

	if year > 0 {
		query += fmt.Sprintf(" AND billing_year = $%d", argIdx)
		args = append(args, year)
		argIdx++
	}
	if month > 0 {
		query += fmt.Sprintf(" AND billing_month = $%d", argIdx)
		args = append(args, month)
		argIdx++
	}
	if service != "" {
		query += fmt.Sprintf(" AND service_type = $%d", argIdx)
		args = append(args, service)
		argIdx++
	}
	query += " ORDER BY billing_year DESC, billing_month DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error al listar recibos principales: %w", err)
	}
	defer rows.Close()

	var bills []domain.MasterBill
	for rows.Next() {
		var b domain.MasterBill
		if err := rows.Scan(
			&b.ID, &b.PropertyID, &b.ServiceType, &b.BillingYear, &b.BillingMonth,
			&b.TotalConsumption, &b.UnitPrice, &b.ConsumptionAmount, &b.ExtraChargesTotal,
			&b.TotalAmount, &b.IssueDate, &b.DueDate, &b.Status, &b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("error al escanear recibo principal: %w", err)
		}
		bills = append(bills, b)
	}

	return bills, nil
}

func (r *MasterBillRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.MasterBillStatus) error {
	query := `
		UPDATE master_bills
		SET status = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	cmdTag, err := r.pool.Exec(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("error al actualizar estado de recibo principal: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
