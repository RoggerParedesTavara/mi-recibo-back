package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RoggerParedesTavara/mi-recibo/internal/config"
	httpHandler "github.com/RoggerParedesTavara/mi-recibo/internal/handler/http"
	"github.com/RoggerParedesTavara/mi-recibo/internal/repository/postgres"
	"github.com/RoggerParedesTavara/mi-recibo/internal/service"
)

func main() {
	log.Println("Iniciando servicio Mi Recibo API...")

	// 1. Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error cargando configuración: %v", err)
	}

	// 2. Database connection pool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := postgres.NewDB(ctx, cfg.DatabaseURL())
	if err != nil {
		log.Fatalf("Error conectando a la base de datos PostgreSQL: %v", err)
	}
	defer db.Close()
	log.Println("Conexión con PostgreSQL 16 establecida exitosamente.")

	// 3. Run database migrations
	migrationsDir := "migrations"
	if err := db.RunMigrations(context.Background(), migrationsDir); err != nil {
		log.Printf("Advertencia en migraciones (pueden ya estar aplicadas): %v", err)
	} else {
		log.Println("Migraciones SQL ejecutadas correctamente.")
	}

	// 4. Initialize Repositories
	ownerRepo := postgres.NewOwnerRepository(db.Pool)
	propertyRepo := postgres.NewPropertyRepository(db.Pool)
	tenantRepo := postgres.NewTenantRepository(db.Pool)
	masterBillRepo := postgres.NewMasterBillRepository(db.Pool)
	consumptionRepo := postgres.NewConsumptionRepository(db.Pool)
	tenantBillRepo := postgres.NewTenantBillRepository(db.Pool)

	// 5. Initialize Services
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

	// 6. Initialize HTTP Handlers & Router
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

	server := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 7. Start server in goroutine
	go func() {
		log.Printf("Servidor HTTP escuchando en http://localhost:%s", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Error en servidor HTTP: %v", err)
		}
	}()

	// 8. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("Señal recibida (%v), cerrando servidor...", sig)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Error forzando cierre del servidor: %v", err)
	}

	fmt.Println("Servidor cerrado ordenadamente.")
}
