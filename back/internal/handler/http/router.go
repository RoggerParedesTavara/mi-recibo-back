package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type RouterParams struct {
	OwnerHandler    *OwnerHandler
	PropertyHandler *PropertyHandler
	TenantHandler   *TenantHandler
	BillHandler     *BillHandler
}

func NewRouter(p RouterParams) http.Handler {
	r := chi.NewRouter()

	// Global middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		RespondJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"service": "mi-recibo-api",
		})
	})

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Owners
		r.Route("/owners", func(r chi.Router) {
			r.Post("/", p.OwnerHandler.CreateOwner)
			r.Get("/{id}", p.OwnerHandler.GetOwner)
			r.Post("/{owner_id}/properties", p.PropertyHandler.CreateProperty)
			r.Get("/{owner_id}/properties", p.PropertyHandler.ListPropertiesByOwner)
		})

		// Properties
		r.Route("/properties", func(r chi.Router) {
			r.Get("/{id}", p.PropertyHandler.GetProperty)
			r.Post("/{property_id}/tenants", p.TenantHandler.CreateTenant)
			r.Get("/{property_id}/tenants", p.TenantHandler.ListTenantsByProperty)
			r.Post("/{property_id}/master-bills", p.BillHandler.CreateMasterBill)
			r.Get("/{property_id}/master-bills", p.BillHandler.ListMasterBills)
		})

		// Tenants
		r.Route("/tenants", func(r chi.Router) {
			r.Put("/{id}", p.TenantHandler.UpdateTenant)
			r.Get("/{id}/bills", p.TenantHandler.ListTenantBills)
		})

		// Master Bills
		r.Route("/master-bills", func(r chi.Router) {
			r.Get("/{id}", p.BillHandler.GetMasterBill)
			r.Post("/{id}/consumptions", p.BillHandler.RecordConsumptions)
			r.Post("/{id}/calculate", p.BillHandler.CalculateMasterBill)
			r.Get("/{id}/tenant-bills", p.BillHandler.ListMasterBillTenantBills)
		})

		// Tenant Bills
		r.Route("/tenant-bills", func(r chi.Router) {
			r.Get("/{id}", p.BillHandler.GetTenantBill)
			r.Patch("/{id}/payment", p.BillHandler.UpdateTenantBillPayment)
		})
	})

	return r
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
