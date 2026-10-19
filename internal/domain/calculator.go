package domain

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CalculationResult holds the calculated bills and itemized charge breakdowns for all active tenants
type CalculationResult struct {
	TenantBills []TenantBill
}

// CalculateBills executes the pure calculation logic:
// 1. Validates that N (number of active tenants) > 0.
// 2. Multiplies each tenant's consumed units strictly by the unit price.
// 3. For each extra charge in the master bill's dynamic list, calculates tenant_share = amount / N.
// 4. Sums consumption amount + total extra charges share for each tenant.
func CalculateBills(
	masterBill *MasterBill,
	activeTenants []Tenant,
	consumptions []TenantConsumption,
) (*CalculationResult, error) {
	n := len(activeTenants)
	if n == 0 {
		return nil, ErrNoActiveTenants
	}

	// Map consumptions by tenant ID
	consumptionMap := make(map[uuid.UUID]TenantConsumption, len(consumptions))
	for _, c := range consumptions {
		consumptionMap[c.TenantID] = c
	}

	// Verify every active tenant has a consumption record
	for _, tenant := range activeTenants {
		if _, ok := consumptionMap[tenant.ID]; !ok {
			return nil, fmt.Errorf("%w: inquilino '%s' (ID: %s)", ErrMissingConsumptions, tenant.Name, tenant.ID)
		}
	}

	divisor := decimal.NewFromInt(int64(n))
	bills := make([]TenantBill, 0, n)

	for _, tenant := range activeTenants {
		c := consumptionMap[tenant.ID]
		consumedUnits := c.ConsumedUnits

		// Variable consumption cost: units * unit_price
		consumptionCost := consumedUnits.Mul(masterBill.UnitPrice).Round(2)

		// Calculate share for each extra charge in the dynamic list
		breakdowns := make([]TenantBillChargeBreakdown, 0, len(masterBill.ExtraCharges))
		extraChargesTotalShare := decimal.Zero

		for _, extra := range masterBill.ExtraCharges {
			// tenant_share = amount / N
			share := extra.Amount.Div(divisor).Round(2)
			extraChargesTotalShare = extraChargesTotalShare.Add(share)

			extraID := extra.ID
			breakdowns = append(breakdowns, TenantBillChargeBreakdown{
				ID:             uuid.New(),
				ExtraChargeID:  &extraID,
				Description:    extra.Description,
				OriginalAmount: extra.Amount,
				TenantsCount:   n,
				TenantShare:    share,
			})
		}

		totalBillAmount := consumptionCost.Add(extraChargesTotalShare)

		bills = append(bills, TenantBill{
			ID:                     uuid.New(),
			MasterBillID:           masterBill.ID,
			TenantID:               tenant.ID,
			ConsumedUnits:          consumedUnits,
			UnitPrice:              masterBill.UnitPrice,
			ConsumptionAmount:      consumptionCost,
			ExtraChargesTotalShare: extraChargesTotalShare,
			TotalAmount:            totalBillAmount,
			Status:                 TenantBillPending,
			Breakdowns:             breakdowns,
		})
	}

	return &CalculationResult{
		TenantBills: bills,
	}, nil
}

// CalculateConsumptionVariation compares the current consumption with the previous month's
// and sets alert flags if it exceeds the specified spike threshold percentage (e.g. 20%).
func CalculateConsumptionVariation(
	currentUnits decimal.Decimal,
	previousMonthUnits *decimal.Decimal,
	spikeThresholdPercent float64,
) (variation *decimal.Decimal, alertSpike bool, alertMessage string) {
	if previousMonthUnits == nil || previousMonthUnits.IsZero() {
		return nil, false, ""
	}

	prev := *previousMonthUnits
	// variation% = ((current - prev) / prev) * 100
	diff := currentUnits.Sub(prev)
	pct := diff.Div(prev).Mul(decimal.NewFromInt(100)).Round(2)
	variation = &pct

	threshold := decimal.NewFromFloat(spikeThresholdPercent)
	if pct.GreaterThan(threshold) {
		alertSpike = true
		alertMessage = fmt.Sprintf(
			"El consumo aumentó un %s%% respecto al mes anterior (de %s a %s unidades).",
			pct.StringFixed(2),
			prev.StringFixed(2),
			currentUnits.StringFixed(2),
		)
	}

	return variation, alertSpike, alertMessage
}
