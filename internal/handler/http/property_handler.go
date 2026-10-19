package http

import (
	"encoding/json"
	"net/http"

	"github.com/RoggerParedesTavara/mi-recibo/internal/domain"
	"github.com/RoggerParedesTavara/mi-recibo/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type PropertyHandler struct {
	propertyService *service.PropertyService
}

func NewPropertyHandler(propertyService *service.PropertyService) *PropertyHandler {
	return &PropertyHandler{propertyService: propertyService}
}

type CreatePropertyRequest struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

func (h *PropertyHandler) CreateProperty(w http.ResponseWriter, r *http.Request) {
	ownerIDStr := chi.URLParam(r, "owner_id")
	ownerID, err := uuid.Parse(ownerIDStr)
	if err != nil {
		RespondError(w, domain.ErrValidation)
		return
	}

	var req CreatePropertyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrValidation)
		return
	}

	property, err := h.propertyService.CreateProperty(r.Context(), ownerID, req.Name, req.Address)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusCreated, property)
}

func (h *PropertyHandler) GetProperty(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		RespondError(w, domain.ErrValidation)
		return
	}

	property, err := h.propertyService.GetProperty(r.Context(), id)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, property)
}

func (h *PropertyHandler) ListPropertiesByOwner(w http.ResponseWriter, r *http.Request) {
	ownerIDStr := chi.URLParam(r, "owner_id")
	ownerID, err := uuid.Parse(ownerIDStr)
	if err != nil {
		RespondError(w, domain.ErrValidation)
		return
	}

	properties, err := h.propertyService.ListPropertiesByOwner(r.Context(), ownerID)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, properties)
}
