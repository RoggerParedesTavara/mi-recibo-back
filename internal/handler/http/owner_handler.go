package http

import (
	"encoding/json"
	"net/http"

	"github.com/RoggerParedesTavara/mi-recibo/internal/domain"
	"github.com/RoggerParedesTavara/mi-recibo/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type OwnerHandler struct {
	propertyService *service.PropertyService
}

func NewOwnerHandler(propertyService *service.PropertyService) *OwnerHandler {
	return &OwnerHandler{propertyService: propertyService}
}

type CreateOwnerRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

func (h *OwnerHandler) CreateOwner(w http.ResponseWriter, r *http.Request) {
	var req CreateOwnerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrValidation)
		return
	}

	owner, err := h.propertyService.CreateOwner(r.Context(), req.Name, req.Email, req.Phone)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusCreated, owner)
}

func (h *OwnerHandler) GetOwner(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		RespondError(w, domain.ErrValidation)
		return
	}

	owner, err := h.propertyService.GetOwner(r.Context(), id)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, owner)
}
