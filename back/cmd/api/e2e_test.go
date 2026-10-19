package main_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/RoggerParedesTavara/mi-recibo/internal/config"
	"github.com/RoggerParedesTavara/mi-recibo/internal/domain"
	httpHandler "github.com/RoggerParedesTavara/mi-recibo/internal/handler/http"
	"github.com/RoggerParedesTavara/mi-recibo/internal/repository/postgres"
	"github.com/RoggerParedesTavara/mi-recibo/internal/service"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func setupTestServer(t *testing.T) (http.Handler, *postgres.DB) {
	_ = os.Setenv("DB_NAME", "mi_recibo")
	_ = os.Setenv("DB_USER", "postgres")
	_ = os.Setenv("DB_PASSWORD", "postgres")
	_ = os.Setenv("DB_HOST", "localhost")
	_ = os.Setenv("DB_PORT", "5432")
	_ = os.Setenv("DB_SSLMODE", "disable")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := postgres.NewDB(ctx, cfg.DatabaseURL())
	if err != nil {
		t.Skipf("skipping integration test: database not available: %v", err)
	}

	// Run migrations
	if err := db.RunMigrations(context.Background(), "../../migrations"); err != nil {
		// Try relative to current working dir
		if err2 := db.RunMigrations(context.Background(), "migrations"); err2 != nil {
			t.Logf("migrations warning: %v / %v", err, err2)
		}
	}

	ownerRepo := postgres.NewOwnerRepository(db.Pool)
	propertyRepo := postgres.NewPropertyRepository(db.Pool)
	tenantRepo := postgres.NewTenantRepository(db.Pool)
	masterBillRepo := postgres.NewMasterBillRepository(db.Pool)
	consumptionRepo := postgres.NewConsumptionRepository(db.Pool)
	tenantBillRepo := postgres.NewTenantBillRepository(db.Pool)

	propertyService := service.NewPropertyService(ownerRepo, propertyRepo)
	tenantService := service.NewTenantService(tenantRepo, propertyRepo)
	billingService := service.NewBillingService(
		masterBillRepo,
		consumptionRepo,
		tenantBillRepo,
		tenantRepo,
		propertyRepo,
		cfg.ConsumptionAlertSpikeThreshold,
	)

	ownerHandler := httpHandler.NewOwnerHandler(propertyService)
	propertyHandler := httpHandler.NewPropertyHandler(propertyService)
	tenantHandler := httpHandler.NewTenantHandler(tenantService, billingService)
	billHandler := httpHandler.NewBillHandler(billingService)

	router := httpHandler.NewRouter(httpHandler.RouterParams{
		OwnerHandler:    ownerHandler,
		PropertyHandler: propertyHandler,
		TenantHandler:   tenantHandler,
		BillHandler:     billHandler,
	})

	return router, db
}

func TestCompleteBillingWorkflow(t *testing.T) {
	router, db := setupTestServer(t)
	defer db.Close()

	// 1. Create Owner
	ownerEmail := fmt.Sprintf("owner_%s@test.com", uuid.New().String()[:8])
	ownerPayload := map[string]string{
		"name":  "Carlos Dueño",
		"email": ownerEmail,
		"phone": "+54911223344",
	}
	body, _ := json.Marshal(ownerPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/owners", bytes.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("failed to create owner: code=%d body=%s", w.Code, w.Body.String())
	}
	var owner domain.Owner
	_ = json.Unmarshal(w.Body.Bytes(), &owner)

	// 2. Create Property
	propPayload := map[string]string{
		"name":    "Casa Los Rosales",
		"address": "Av. Principal 123",
	}
	body, _ = json.Marshal(propPayload)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/owners/%s/properties", owner.ID), bytes.NewReader(body))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("failed to create property: code=%d body=%s", w.Code, w.Body.String())
	}
	var property domain.Property
	_ = json.Unmarshal(w.Body.Bytes(), &property)

	// 3. Create 3 Tenants
	tenants := make([]domain.Tenant, 3)
	for i := 0; i < 3; i++ {
		tPayload := map[string]string{
			"name":         fmt.Sprintf("Inquilino %d", i+1),
			"email":        fmt.Sprintf("inquilino%d_%s@test.com", i+1, uuid.New().String()[:6]),
			"phone":        "+54911000000",
			"room_or_unit": fmt.Sprintf("Habitación %d", i+1),
		}
		body, _ = json.Marshal(tPayload)
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/properties/%s/tenants", property.ID), bytes.NewReader(body))
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("failed to create tenant %d: code=%d body=%s", i+1, w.Code, w.Body.String())
		}
		_ = json.Unmarshal(w.Body.Bytes(), &tenants[i])
	}

	// 4. Create Master Bill for September 2026 (to seed previous month consumption for tenant 1)
	septBillPayload := map[string]any{
		"service_type":      "LUZ",
		"billing_year":      2026,
		"billing_month":     9,
		"total_consumption": 200.0,
		"unit_price":        0.50,
		"extra_charges":     []any{},
	}
	body, _ = json.Marshal(septBillPayload)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/properties/%s/master-bills", property.ID), bytes.NewReader(body))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var septBill domain.MasterBill
	_ = json.Unmarshal(w.Body.Bytes(), &septBill)

	// Seed September readings: Tenant 1 consumed 60.0 units
	septReadings := map[string]any{
		"readings": []any{
			map[string]any{"tenant_id": tenants[0].ID, "consumed_units": 60.0},
			map[string]any{"tenant_id": tenants[1].ID, "consumed_units": 100.0},
			map[string]any{"tenant_id": tenants[2].ID, "consumed_units": 90.0},
		},
	}
	body, _ = json.Marshal(septReadings)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/master-bills/%s/consumptions", septBill.ID), bytes.NewReader(body))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("failed to record sept consumptions: code=%d body=%s", w.Code, w.Body.String())
	}

	// 5. Create Master Bill for October 2026 with Dynamic Extra Charges List
	octBillPayload := map[string]any{
		"service_type":      "LUZ",
		"billing_year":      2026,
		"billing_month":     10,
		"total_consumption": 350.0,
		"unit_price":        0.50,
		"extra_charges": []map[string]any{
			{"description": "Mantenimiento de la red", "amount": 300.00},
			{"description": "Alumbrado público", "amount": 45.00},
			{"description": "Cargo fijo", "amount": 12.00},
		},
	}
	body, _ = json.Marshal(octBillPayload)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/properties/%s/master-bills", property.ID), bytes.NewReader(body))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("failed to create oct master bill: code=%d body=%s", w.Code, w.Body.String())
	}
	var octBill domain.MasterBill
	_ = json.Unmarshal(w.Body.Bytes(), &octBill)

	// Verify extra charges total on master bill: 300 + 45 + 12 = 357.00
	expectedExtraTotal := decimal.NewFromFloat(357.00)
	if !octBill.ExtraChargesTotal.Equal(expectedExtraTotal) {
		t.Errorf("expected extra charges total %s, got %s", expectedExtraTotal, octBill.ExtraChargesTotal)
	}

	// 6. Record Consumptions for October 2026
	// Tenant 1 consumes 80.0 units (+33.33% vs 60 in Sept -> spike alert)
	// Tenant 2 consumes 120.0 units
	// Tenant 3 consumes 100.0 units
	octReadings := map[string]any{
		"readings": []any{
			map[string]any{"tenant_id": tenants[0].ID, "consumed_units": 80.0},
			map[string]any{"tenant_id": tenants[1].ID, "consumed_units": 120.0},
			map[string]any{"tenant_id": tenants[2].ID, "consumed_units": 100.0},
		},
	}
	body, _ = json.Marshal(octReadings)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/master-bills/%s/consumptions", octBill.ID), bytes.NewReader(body))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("failed to record oct consumptions: code=%d body=%s", w.Code, w.Body.String())
	}

	// 7. Calculate Master Bill
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/master-bills/%s/calculate", octBill.ID), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("failed to calculate bills: code=%d body=%s", w.Code, w.Body.String())
	}

	var calcResponse struct {
		Message     string                    `json:"message"`
		TenantBills []domain.TenantBillDetail `json:"tenant_bills"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &calcResponse)

	if len(calcResponse.TenantBills) != 3 {
		t.Fatalf("expected 3 tenant bills, got %d", len(calcResponse.TenantBills))
	}

	// Find bill for Tenant 1
	var b1 *domain.TenantBillDetail
	for i := range calcResponse.TenantBills {
		if calcResponse.TenantBills[i].TenantID == tenants[0].ID {
			b1 = &calcResponse.TenantBills[i]
			break
		}
	}
	if b1 == nil {
		t.Fatal("bill for tenant 1 not found")
	}

	// Verify Tenant 1 calculations:
	// Consumption: 80 * 0.50 = 40.00
	// Extra charges: (300/3) + (45/3) + (12/3) = 100 + 15 + 4 = 119.00
	// Total: 40 + 119 = 159.00
	if !b1.ConsumptionAmount.Equal(decimal.NewFromFloat(40.00)) {
		t.Errorf("tenant 1 expected consumption amount 40.00, got %s", b1.ConsumptionAmount)
	}
	if !b1.ExtraChargesTotalShare.Equal(decimal.NewFromFloat(119.00)) {
		t.Errorf("tenant 1 expected extra charges share 119.00, got %s", b1.ExtraChargesTotalShare)
	}
	if !b1.TotalAmount.Equal(decimal.NewFromFloat(159.00)) {
		t.Errorf("tenant 1 expected total amount 159.00, got %s", b1.TotalAmount)
	}

	// Verify Spike Alert:
	if !b1.AlertSpike {
		t.Errorf("expected tenant 1 to have alert_spike=true")
	}
	if b1.AlertMessage == "" {
		t.Errorf("expected tenant 1 to have non-empty alert_message")
	}

	// Verify Breakdown items
	if len(b1.Breakdowns) != 3 {
		t.Fatalf("expected 3 breakdown items for tenant 1, got %d", len(b1.Breakdowns))
	}
	if b1.Breakdowns[0].Description != "Mantenimiento de la red" || !b1.Breakdowns[0].TenantShare.Equal(decimal.NewFromFloat(100.00)) {
		t.Errorf("breakdown 0 mismatch: %+v", b1.Breakdowns[0])
	}
	if b1.Breakdowns[1].Description != "Alumbrado público" || !b1.Breakdowns[1].TenantShare.Equal(decimal.NewFromFloat(15.00)) {
		t.Errorf("breakdown 1 mismatch: %+v", b1.Breakdowns[1])
	}
	if b1.Breakdowns[2].Description != "Cargo fijo" || !b1.Breakdowns[2].TenantShare.Equal(decimal.NewFromFloat(4.00)) {
		t.Errorf("breakdown 2 mismatch: %+v", b1.Breakdowns[2])
	}

	// 8. Test marking bill as paid
	payPayload := map[string]bool{"is_paid": true}
	body, _ = json.Marshal(payPayload)
	req = httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/tenant-bills/%s/payment", b1.ID), bytes.NewReader(body))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("failed to update payment: code=%d body=%s", w.Code, w.Body.String())
	}

	// Verify bill is now PAID
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/tenant-bills/%s", b1.ID), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var updatedBill domain.TenantBillDetail
	_ = json.Unmarshal(w.Body.Bytes(), &updatedBill)
	if updatedBill.Status != domain.TenantBillPaid {
		t.Errorf("expected status PAID, got %s", updatedBill.Status)
	}
	if updatedBill.PaidAt == nil {
		t.Errorf("expected paid_at timestamp to be set")
	}
}
