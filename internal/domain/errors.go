package domain

import "errors"

var (
	ErrNotFound            = errors.New("recurso no encontrado")
	ErrConflict            = errors.New("conflicto de datos o registro duplicado")
	ErrValidation          = errors.New("error de validación")
	ErrNoActiveTenants     = errors.New("la propiedad no tiene inquilinos activos para calcular el recibo")
	ErrMissingConsumptions = errors.New("faltan lecturas de consumo para uno o más inquilinos activos")
	ErrAlreadyCalculated   = errors.New("el recibo principal ya ha sido liquidado")
)
