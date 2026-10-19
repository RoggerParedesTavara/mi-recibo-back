package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type TenantBillStatus string

const (
	TenantBillPending TenantBillStatus = "PENDING"
	TenantBillPaid    TenantBillStatus = "PAID"
)

// TenantBillChargeBreakdown represents an itemized breakdown for each extra charge split among tenants
type TenantBillChargeBreakdown struct {
	ID             uuid.UUID       `json:"id"`
	TenantBillID   uuid.UUID       `json:"tenant_bill_id"`
	ExtraChargeID  *uuid.UUID      `json:"extra_charge_id,omitempty"`
	Description    string          `json:"description"`
	OriginalAmount decimal.Decimal `json:"original_amount"` // Total charge amount for the entire property
	TenantsCount   int             `json:"tenants_count"`   // Divisor N
	TenantShare    decimal.Decimal `json:"tenant_share"`    // OriginalAmount / N
	CreatedAt      time.Time       `json:"created_at"`
}

// TenantBill is the final monthly bill issued to a single tenant
type TenantBill struct {
	ID                     uuid.UUID                   `json:"id"`
	MasterBillID           uuid.UUID                   `json:"master_bill_id"`
	TenantID               uuid.UUID                   `json:"tenant_id"`
	ConsumedUnits          decimal.Decimal             `json:"consumed_units"`
	UnitPrice              decimal.Decimal             `json:"unit_price"`
	ConsumptionAmount      decimal.Decimal             `json:"consumption_amount"`
	ExtraChargesTotalShare decimal.Decimal             `json:"extra_charges_total_share"`
	TotalAmount            decimal.Decimal             `json:"total_amount"`
	Status                 TenantBillStatus            `json:"status"`
	PaidAt                 *time.Time                  `json:"paid_at,omitempty"`
	Breakdowns             []TenantBillChargeBreakdown `json:"breakdowns,omitempty"`
	CreatedAt              time.Time                   `json:"created_at"`
	UpdatedAt              time.Time                   `json:"updated_at"`
}

// TenantBillDetail is an enriched view for presentation in API and MCP
type TenantBillDetail struct {
	TenantBill
	TenantName         string           `json:"tenant_name"`
	RoomOrUnit         string           `json:"room_or_unit,omitempty"`
	PropertyName       string           `json:"property_name"`
	ServiceType        ServiceType      `json:"service_type"`
	BillingYear        int              `json:"billing_year"`
	BillingMonth       int              `json:"billing_month"`
	AlertSpike         bool             `json:"alert_spike"`
	AlertMessage       string           `json:"alert_message,omitempty"`
	PreviousMonthUnits *decimal.Decimal `json:"previous_month_units,omitempty"`
}
