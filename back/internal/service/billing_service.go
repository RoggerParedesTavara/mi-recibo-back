package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/RoggerParedesTavara/mi-recibo/internal/domain"
	"github.com/RoggerParedesTavara/mi-recibo/internal/repository/postgres"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type CreateMasterBillInput struct {
	PropertyID       uuid.UUID                 `json:"property_id"`
	ServiceType      domain.ServiceType        `json:"service_type"`
	BillingYear      int                       `json:"billing_year"`
	BillingMonth     int                       `json:"billing_month"`
	TotalConsumption decimal.Decimal           `json:"total_consumption"`
	UnitPrice        decimal.Decimal           `json:"unit_price"`
	IssueDate        *time.Time                `json:"issue_date,omitempty"`
	DueDate          *time.Time                `json:"due_date,omitempty"`
	ExtraCharges     []CreateExtraChargeInput  `json:"extra_charges"`
}

type CreateExtraChargeInput struct {
	Description string          `json:"description"`
	Amount      decimal.Decimal `json:"amount"`
}

type BillingService struct {
	masterBillRepo        *postgres.MasterBillRepository
	consumptionRepo       *postgres.ConsumptionRepository
	tenantBillRepo        *postgres.TenantBillRepository
	tenantRepo            *postgres.TenantRepository
	propertyRepo          *postgres.PropertyRepository
	spikeThresholdPercent float64
}

func NewBillingService(
	masterBillRepo *postgres.MasterBillRepository,
	consumptionRepo *postgres.ConsumptionRepository,
	tenantBillRepo *postgres.TenantBillRepository,
	tenantRepo *postgres.TenantRepository,
	propertyRepo *postgres.PropertyRepository,
	spikeThresholdPercent float64,
) *BillingService {
	return &BillingService{
		masterBillRepo:        masterBillRepo,
		consumptionRepo:       consumptionRepo,
		tenantBillRepo:        tenantBillRepo,
		tenantRepo:            tenantRepo,
		propertyRepo:          propertyRepo,
		spikeThresholdPercent: spikeThresholdPercent,
	}
}

func (s *BillingService) CreateMasterBill(ctx context.Context, input CreateMasterBillInput) (*domain.MasterBill, error) {
	// Validate service type
	switch input.ServiceType {
	case domain.ServiceLuz, domain.ServiceAgua, domain.ServiceGas:
	default:
		return nil, fmt.Errorf("%w: tipo de servicio inválido (debe ser LUZ, AGUA o GAS)", domain.ErrValidation)
	}

	if input.BillingMonth < 1 || input.BillingMonth > 12 || input.BillingYear < 2000 {
		return nil, fmt.Errorf("%w: año o mes de facturación inválido", domain.ErrValidation)
	}

	if input.UnitPrice.IsNegative() || input.UnitPrice.IsZero() {
		return nil, fmt.Errorf("%w: el precio por unidad debe ser mayor a 0", domain.ErrValidation)
	}

	if input.TotalConsumption.IsNegative() {
		return nil, fmt.Errorf("%w: el consumo total no puede ser negativo", domain.ErrValidation)
	}

	// Verify property exists
	if _, err := s.propertyRepo.GetByID(ctx, input.PropertyID); err != nil {
		return nil, err
	}

	// Map dynamic extra charges
	extraCharges := make([]domain.MasterBillExtraCharge, 0, len(input.ExtraCharges))
	for _, ec := range input.ExtraCharges {
		desc := strings.TrimSpace(ec.Description)
		if desc == "" {
			return nil, fmt.Errorf("%w: cada cargo extra debe tener una descripción", domain.ErrValidation)
		}
		if ec.Amount.IsNegative() {
			return nil, fmt.Errorf("%w: el monto del cargo extra '%s' no puede ser negativo", domain.ErrValidation, desc)
		}
		extraCharges = append(extraCharges, domain.MasterBillExtraCharge{
			Description: desc,
			Amount:      ec.Amount,
		})
	}

	bill := &domain.MasterBill{
		PropertyID:       input.PropertyID,
		ServiceType:      input.ServiceType,
		BillingYear:      input.BillingYear,
		BillingMonth:     input.BillingMonth,
		TotalConsumption: input.TotalConsumption,
		UnitPrice:        input.UnitPrice,
		IssueDate:        input.IssueDate,
		DueDate:          input.DueDate,
		ExtraCharges:     extraCharges,
	}

	if err := s.masterBillRepo.Create(ctx, bill); err != nil {
		return nil, err
	}

	return bill, nil
}

func (s *BillingService) GetMasterBill(ctx context.Context, id uuid.UUID) (*domain.MasterBill, error) {
	return s.masterBillRepo.GetByID(ctx, id)
}

func (s *BillingService) ListMasterBills(ctx context.Context, propertyID uuid.UUID, year, month int, service domain.ServiceType) ([]domain.MasterBill, error) {
	return s.masterBillRepo.ListByProperty(ctx, propertyID, year, month, service)
}

func (s *BillingService) RecordConsumptions(ctx context.Context, masterBillID uuid.UUID, readings []domain.ReadingInput) ([]domain.TenantConsumption, error) {
	if len(readings) == 0 {
		return nil, fmt.Errorf("%w: no se proporcionaron lecturas", domain.ErrValidation)
	}

	masterBill, err := s.masterBillRepo.GetByID(ctx, masterBillID)
	if err != nil {
		return nil, err
	}

	consumptions := make([]domain.TenantConsumption, 0, len(readings))
	for _, r := range readings {
		if r.ConsumedUnits.IsNegative() {
			return nil, fmt.Errorf("%w: el consumo no puede ser negativo para inquilino %s", domain.ErrValidation, r.TenantID)
		}

		// Look up previous month consumption
		prevUnits, err := s.consumptionRepo.GetPreviousMonthConsumption(
			ctx, r.TenantID, masterBill.ServiceType, masterBill.BillingYear, masterBill.BillingMonth,
		)
		if err != nil {
			return nil, err
		}

		variation, alertSpike, alertMessage := domain.CalculateConsumptionVariation(
			r.ConsumedUnits, prevUnits, s.spikeThresholdPercent,
		)

		consumptions = append(consumptions, domain.TenantConsumption{
			MasterBillID:             masterBillID,
			TenantID:                 r.TenantID,
			ConsumedUnits:            r.ConsumedUnits,
			PreviousReading:          r.PreviousReading,
			CurrentReading:           r.CurrentReading,
			PreviousMonthConsumption: prevUnits,
			VariationPercentage:      variation,
			AlertSpike:               alertSpike,
			AlertMessage:             alertMessage,
		})
	}

	if err := s.consumptionRepo.UpsertBatch(ctx, consumptions); err != nil {
		return nil, err
	}

	return consumptions, nil
}

func (s *BillingService) CalculateMasterBill(ctx context.Context, masterBillID uuid.UUID) ([]domain.TenantBillDetail, error) {
	masterBill, err := s.masterBillRepo.GetByID(ctx, masterBillID)
	if err != nil {
		return nil, err
	}

	// Fetch active tenants in the property
	activeTenants, err := s.tenantRepo.ListByPropertyID(ctx, masterBill.PropertyID, true)
	if err != nil {
		return nil, err
	}

	// Fetch recorded consumptions
	consumptions, err := s.consumptionRepo.ListByMasterBillID(ctx, masterBillID)
	if err != nil {
		return nil, err
	}

	// Pure domain calculation
	calcResult, err := domain.CalculateBills(masterBill, activeTenants, consumptions)
	if err != nil {
		return nil, err
	}

	// Persist calculation result
	if err := s.tenantBillRepo.SaveCalculationResult(ctx, masterBillID, calcResult.TenantBills); err != nil {
		return nil, err
	}

	// Return enriched bill details
	return s.tenantBillRepo.ListByMasterBillID(ctx, masterBillID)
}

func (s *BillingService) GetTenantBill(ctx context.Context, id uuid.UUID) (*domain.TenantBillDetail, error) {
	return s.tenantBillRepo.GetByID(ctx, id)
}

func (s *BillingService) ListTenantBillsByMasterBill(ctx context.Context, masterBillID uuid.UUID) ([]domain.TenantBillDetail, error) {
	return s.tenantBillRepo.ListByMasterBillID(ctx, masterBillID)
}

func (s *BillingService) ListTenantBillsByTenant(ctx context.Context, tenantID uuid.UUID) ([]domain.TenantBillDetail, error) {
	return s.tenantBillRepo.ListByTenantID(ctx, tenantID)
}

func (s *BillingService) UpdateTenantBillPayment(ctx context.Context, id uuid.UUID, isPaid bool) error {
	status := domain.TenantBillPending
	if isPaid {
		status = domain.TenantBillPaid
	}
	return s.tenantBillRepo.UpdatePaymentStatus(ctx, id, status)
}
