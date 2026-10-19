package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ServiceType string

const (
	ServiceLuz  ServiceType = "LUZ"
	ServiceAgua ServiceType = "AGUA"
	ServiceGas  ServiceType = "GAS"
)

type MasterBillStatus string

const (
	BillStatusDraft      MasterBillStatus = "DRAFT"
	BillStatusCalculated MasterBillStatus = "CALCULATED"
	BillStatusPaid       MasterBillStatus = "PAID"
)

// MasterBillExtraCharge represents an individual extra fee/tax on the master bill (e.g. "Mantenimiento de red", "Alumbrado")
type MasterBillExtraCharge struct {
	ID           uuid.UUID       `json:"id"`
	MasterBillID uuid.UUID       `json:"master_bill_id"`
	Description  string          `json:"description"`
	Amount       decimal.Decimal `json:"amount"` // Total amount charged for the entire house
	CreatedAt    time.Time       `json:"created_at"`
}

// MasterBill represents the utility bill received by the property/owner for a specific month
type MasterBill struct {
	ID                uuid.UUID               `json:"id"`
	PropertyID        uuid.UUID               `json:"property_id"`
	ServiceType       ServiceType             `json:"service_type"`
	BillingYear       int                     `json:"billing_year"`
	BillingMonth      int                     `json:"billing_month"`
	TotalConsumption  decimal.Decimal         `json:"total_consumption"`  // Total units measured on the main bill
	UnitPrice         decimal.Decimal         `json:"unit_price"`         // Price per unit consumed
	ConsumptionAmount decimal.Decimal         `json:"consumption_amount"` // TotalConsumption * UnitPrice
	ExtraChargesTotal decimal.Decimal         `json:"extra_charges_total"`// Sum of all items in ExtraCharges
	TotalAmount       decimal.Decimal         `json:"total_amount"`       // ConsumptionAmount + ExtraChargesTotal
	IssueDate         *time.Time              `json:"issue_date,omitempty"`
	DueDate           *time.Time              `json:"due_date,omitempty"`
	Status            MasterBillStatus        `json:"status"`
	ExtraCharges      []MasterBillExtraCharge `json:"extra_charges"` // Dynamic list of extra charges
	CreatedAt         time.Time               `json:"created_at"`
	UpdatedAt         time.Time               `json:"updated_at"`
}
