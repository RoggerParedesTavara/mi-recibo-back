package http

import (
	"encoding/json"
	"net/http"

	"github.com/RoggerParedesTavara/mi-recibo/internal/domain"
	"github.com/RoggerParedesTavara/mi-recibo/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TenantHandler struct {
	tenantService  *service.TenantService
	billingService *service.BillingService
}

func NewTenantHandler(tenantService *service.TenantService, billingService *service.BillingService) *TenantHandler {
	return &TenantHandler{
		tenantService:  tenantService,
		billingService: billingService,
	}
}

type CreateTenantRequest struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	RoomOrUnit string `json:"room_or_unit"`
}

func (h *TenantHandler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	propertyIDStr := chi.URLParam(r, "property_id")
	propertyID, err := uuid.Parse(propertyIDStr)
	if err != nil {
		RespondError(w, domain.ErrValidation)
		return
	}

	var req CreateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrValidation)
		return
	}

	tenant, err := h.tenantService.CreateTenant(r.Context(), propertyID, req.Name, req.Email, req.Phone, req.RoomOrUnit)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusCreated, tenant)
}

func (h *TenantHandler) ListTenantsByProperty(w http.ResponseWriter, r *http.Request) {
	propertyIDStr := chi.URLParam(r, "property_id")
	propertyID, err := uuid.Parse(propertyIDStr)
	if err != nil {
		RespondError(w, domain.ErrValidation)
		return
	}

	onlyActive := r.URL.Query().Get("active") == "true"
	tenants, err := h.tenantService.ListTenantsByProperty(r.Context(), propertyID, onlyActive)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, tenants)
}

func (h *TenantHandler) UpdateTenant(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		RespondError(w, domain.ErrValidation)
		return
	}

	var req domain.Tenant
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrValidation)
		return
	}
	req.ID = id

	updated, err := h.tenantService.UpdateTenant(r.Context(), &req)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, updated)
}

func (h *TenantHandler) ListTenantBills(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		RespondError(w, domain.ErrValidation)
		return
	}

	bills, err := h.billingService.ListTenantBillsByTenant(r.Context(), id)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, bills)
}
