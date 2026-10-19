package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/RoggerParedesTavara/mi-recibo/internal/domain"
	"github.com/RoggerParedesTavara/mi-recibo/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type BillHandler struct {
	billingService *service.BillingService
}

func NewBillHandler(billingService *service.BillingService) *BillHandler {
	return &BillHandler{billingService: billingService}
}

type CreateMasterBillRequest struct {
	ServiceType      domain.ServiceType               `json:"service_type"`
	BillingYear      int                              `json:"billing_year"`
	BillingMonth     int                              `json:"billing_month"`
	TotalConsumption decimal.Decimal                  `json:"total_consumption"`
	UnitPrice        decimal.Decimal                  `json:"unit_price"`
	IssueDate        *time.Time                       `json:"issue_date,omitempty"`
	DueDate          *time.Time                       `json:"due_date,omitempty"`
	ExtraCharges     []service.CreateExtraChargeInput `json:"extra_charges"`
}

func (h *BillHandler) CreateMasterBill(w http.ResponseWriter, r *http.Request) {
	propertyIDStr := chi.URLParam(r, "property_id")
	propertyID, err := uuid.Parse(propertyIDStr)
	if err != nil {
		RespondError(w, domain.ErrValidation)
		return
	}

	var req CreateMasterBillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrValidation)
		return
	}

	bill, err := h.billingService.CreateMasterBill(r.Context(), service.CreateMasterBillInput{
		PropertyID:       propertyID,
		ServiceType:      req.ServiceType,
		BillingYear:      req.BillingYear,
		BillingMonth:     req.BillingMonth,
		TotalConsumption: req.TotalConsumption,
		UnitPrice:        req.UnitPrice,
		IssueDate:        req.IssueDate,
		DueDate:          req.DueDate,
		ExtraCharges:     req.ExtraCharges,
	})
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusCreated, bill)
}

func (h *BillHandler) GetMasterBill(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		RespondError(w, domain.ErrValidation)
		return
	}

	bill, err := h.billingService.GetMasterBill(r.Context(), id)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, bill)
}

func (h *BillHandler) ListMasterBills(w http.ResponseWriter, r *http.Request) {
	propertyIDStr := chi.URLParam(r, "property_id")
	propertyID, err := uuid.Parse(propertyIDStr)
	if err != nil {
		RespondError(w, domain.ErrValidation)
		return
	}

	year, _ := strconv.Atoi(r.URL.Query().Get("year"))
	month, _ := strconv.Atoi(r.URL.Query().Get("month"))
	serviceType := domain.ServiceType(r.URL.Query().Get("service"))

	bills, err := h.billingService.ListMasterBills(r.Context(), propertyID, year, month, serviceType)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, bills)
}

type RecordConsumptionsRequest struct {
	Readings []domain.ReadingInput `json:"readings"`
}

func (h *BillHandler) RecordConsumptions(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	masterBillID, err := uuid.Parse(idStr)
	if err != nil {
		RespondError(w, domain.ErrValidation)
		return
	}

	var req RecordConsumptionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrValidation)
		return
	}

	consumptions, err := h.billingService.RecordConsumptions(r.Context(), masterBillID, req.Readings)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, map[string]any{
		"message":      "Lecturas registradas exitosamente",
		"consumptions": consumptions,
	})
}

func (h *BillHandler) CalculateMasterBill(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	masterBillID, err := uuid.Parse(idStr)
	if err != nil {
		RespondError(w, domain.ErrValidation)
		return
	}

	bills, err := h.billingService.CalculateMasterBill(r.Context(), masterBillID)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, map[string]any{
		"message":      "Recibos calculados y generados exitosamente",
		"tenant_bills": bills,
	})
}

func (h *BillHandler) ListMasterBillTenantBills(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	masterBillID, err := uuid.Parse(idStr)
	if err != nil {
		RespondError(w, domain.ErrValidation)
		return
	}

	bills, err := h.billingService.ListTenantBillsByMasterBill(r.Context(), masterBillID)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, bills)
}

func (h *BillHandler) GetTenantBill(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		RespondError(w, domain.ErrValidation)
		return
	}

	bill, err := h.billingService.GetTenantBill(r.Context(), id)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, bill)
}

type UpdatePaymentRequest struct {
	IsPaid bool `json:"is_paid"`
}

func (h *BillHandler) UpdateTenantBillPayment(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		RespondError(w, domain.ErrValidation)
		return
	}

	var req UpdatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrValidation)
		return
	}

	if err := h.billingService.UpdateTenantBillPayment(r.Context(), id, req.IsPaid); err != nil {
		RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, map[string]string{
		"message": "Estado de pago actualizado",
	})
}
