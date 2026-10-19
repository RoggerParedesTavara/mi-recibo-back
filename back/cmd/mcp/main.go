package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/RoggerParedesTavara/mi-recibo/internal/config"
	mcpHandler "github.com/RoggerParedesTavara/mi-recibo/internal/handler/mcp"
	"github.com/RoggerParedesTavara/mi-recibo/internal/repository/postgres"
	"github.com/RoggerParedesTavara/mi-recibo/internal/service"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error cargando configuración: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := postgres.NewDB(ctx, cfg.DatabaseURL())
	if err != nil {
		log.Fatalf("Error conectando a PostgreSQL: %v", err)
	}
	defer db.Close()

	// Repositories
	ownerRepo := postgres.NewOwnerRepository(db.Pool)
	propertyRepo := postgres.NewPropertyRepository(db.Pool)
	tenantRepo := postgres.NewTenantRepository(db.Pool)
	masterBillRepo := postgres.NewMasterBillRepository(db.Pool)
	consumptionRepo := postgres.NewConsumptionRepository(db.Pool)
	tenantBillRepo := postgres.NewTenantBillRepository(db.Pool)

	// Services
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

	// Create MCP Server
	s := server.NewMCPServer("mi-recibo-mcp", "1.0.0")

	// Register tools
	mcpHandler.RegisterMCPTools(s, mcpHandler.ServerParams{
		PropertyService: propertyService,
		TenantService:   tenantService,
		BillingService:  billingService,
	})

	// Start Stdio Server
	stdioServer := server.NewStdioServer(s)
	if err := stdioServer.Listen(context.Background(), os.Stdin, os.Stdout); err != nil {
		log.Fatalf("Error ejecutando servidor MCP stdio: %v", err)
	}
}
