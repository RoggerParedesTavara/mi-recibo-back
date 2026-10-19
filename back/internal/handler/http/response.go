package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/RoggerParedesTavara/mi-recibo/internal/domain"
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

func RespondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

func RespondError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError

	switch {
	case errors.Is(err, domain.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, domain.ErrConflict):
		status = http.StatusConflict
	case errors.Is(err, domain.ErrValidation):
		status = http.StatusBadRequest
	case errors.Is(err, domain.ErrNoActiveTenants):
		status = http.StatusBadRequest
	case errors.Is(err, domain.ErrMissingConsumptions):
		status = http.StatusBadRequest
	case errors.Is(err, domain.ErrAlreadyCalculated):
		status = http.StatusConflict
	}

	RespondJSON(w, status, ErrorResponse{
		Error: err.Error(),
	})
}
