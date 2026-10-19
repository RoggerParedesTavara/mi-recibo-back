package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// TenantConsumption holds the monthly consumed units recorded for a tenant
type TenantConsumption struct {
	ID                       uuid.UUID        `json:"id"`
	MasterBillID             uuid.UUID        `json:"master_bill_id"`
	TenantID                 uuid.UUID        `json:"tenant_id"`
	ConsumedUnits            decimal.Decimal  `json:"consumed_units"`
	PreviousReading          *decimal.Decimal `json:"previous_reading,omitempty"`
	CurrentReading           *decimal.Decimal `json:"current_reading,omitempty"`
	PreviousMonthConsumption *decimal.Decimal `json:"previous_month_consumption,omitempty"`
	VariationPercentage      *decimal.Decimal `json:"variation_percentage,omitempty"`
	AlertSpike               bool             `json:"alert_spike"`
	AlertMessage             string           `json:"alert_message,omitempty"`
	CreatedAt                time.Time        `json:"created_at"`
	UpdatedAt                time.Time        `json:"updated_at"`
}

type ReadingInput struct {
	TenantID        uuid.UUID        `json:"tenant_id"`
	ConsumedUnits   decimal.Decimal  `json:"consumed_units"`
	PreviousReading *decimal.Decimal `json:"previous_reading,omitempty"`
	CurrentReading  *decimal.Decimal `json:"current_reading,omitempty"`
}
