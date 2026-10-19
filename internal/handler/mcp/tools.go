package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/RoggerParedesTavara/mi-recibo/internal/domain"
	"github.com/RoggerParedesTavara/mi-recibo/internal/service"
	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/shopspring/decimal"
)

type ServerParams struct {
	PropertyService *service.PropertyService
	TenantService   *service.TenantService
	BillingService  *service.BillingService
}

func RegisterMCPTools(s *server.MCPServer, params ServerParams) {
	// Tool 1: list_properties
	s.AddTool(
		mcp.NewTool(
			"list_properties",
			mcp.WithDescription("Lista las casas o propiedades pertenecientes a un dueño específico."),
			mcp.WithString("owner_id", mcp.Required(), mcp.Description("UUID del dueño")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ownerIDStr, err := request.RequireString("owner_id")
			if err != nil {
				return mcp.NewToolResultError("owner_id es requerido"), nil
			}
			ownerID, err := uuid.Parse(ownerIDStr)
			if err != nil {
				return mcp.NewToolResultError("UUID de owner_id inválido"), nil
			}

			properties, err := params.PropertyService.ListPropertiesByOwner(ctx, ownerID)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			data, _ := json.MarshalIndent(properties, "", "  ")
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// Tool 2: list_tenants
	s.AddTool(
		mcp.NewTool(
			"list_tenants",
			mcp.WithDescription("Lista los inquilinos de una propiedad con su estado activo/inactivo."),
			mcp.WithString("property_id", mcp.Required(), mcp.Description("UUID de la propiedad")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			propertyIDStr, err := request.RequireString("property_id")
			if err != nil {
				return mcp.NewToolResultError("property_id es requerido"), nil
			}
			propertyID, err := uuid.Parse(propertyIDStr)
			if err != nil {
				return mcp.NewToolResultError("UUID de property_id inválido"), nil
			}

			tenants, err := params.TenantService.ListTenantsByProperty(ctx, propertyID, false)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			data, _ := json.MarshalIndent(tenants, "", "  ")
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// Tool 3: get_master_bill
	s.AddTool(
		mcp.NewTool(
			"get_master_bill",
			mcp.WithDescription("Obtiene los datos del recibo principal del mes (luz/agua/gas), incluyendo su consumo total, tarifa por unidad y su lista de cargos extras/impuestos."),
			mcp.WithString("master_bill_id", mcp.Required(), mcp.Description("UUID del recibo principal")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			idStr, err := request.RequireString("master_bill_id")
			if err != nil {
				return mcp.NewToolResultError("master_bill_id es requerido"), nil
			}
			id, err := uuid.Parse(idStr)
			if err != nil {
				return mcp.NewToolResultError("UUID de master_bill_id inválido"), nil
			}

			bill, err := params.BillingService.GetMasterBill(ctx, id)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			data, _ := json.MarshalIndent(bill, "", "  ")
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// Tool 4: record_tenant_reading
	s.AddTool(
		mcp.NewTool(
			"record_tenant_reading",
			mcp.WithDescription("Registra las unidades consumidas en el mes por un inquilino. Compara con el mes previo y alerta si hubo un incremento inusual."),
			mcp.WithString("master_bill_id", mcp.Required(), mcp.Description("UUID del recibo principal")),
			mcp.WithString("tenant_id", mcp.Required(), mcp.Description("UUID del inquilino")),
			mcp.WithNumber("consumed_units", mcp.Required(), mcp.Description("Unidades consumidas")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			mbIDStr, err := request.RequireString("master_bill_id")
			if err != nil {
				return mcp.NewToolResultError("master_bill_id es requerido"), nil
			}
			tenantIDStr, err := request.RequireString("tenant_id")
			if err != nil {
				return mcp.NewToolResultError("tenant_id es requerido"), nil
			}
			unitsVal, err := request.RequireFloat("consumed_units")
			if err != nil {
				return mcp.NewToolResultError("consumed_units es requerido y debe ser número"), nil
			}

			mbID, err := uuid.Parse(mbIDStr)
			if err != nil {
				return mcp.NewToolResultError("UUID master_bill_id inválido"), nil
			}
			tenantID, err := uuid.Parse(tenantIDStr)
			if err != nil {
				return mcp.NewToolResultError("UUID tenant_id inválido"), nil
			}

			readings := []domain.ReadingInput{
				{
					TenantID:      tenantID,
					ConsumedUnits: decimal.NewFromFloat(unitsVal),
				},
			}

			res, err := params.BillingService.RecordConsumptions(ctx, mbID, readings)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			data, _ := json.MarshalIndent(res, "", "  ")
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// Tool 5: calculate_master_bill
	s.AddTool(
		mcp.NewTool(
			"calculate_master_bill",
			mcp.WithDescription("Ejecuta la liquidación del recibo principal: calcula los recibos de todos los inquilinos activos prorrateando equitativamente los cargos extras y aplicando el consumo estricto de cada uno."),
			mcp.WithString("master_bill_id", mcp.Required(), mcp.Description("UUID del recibo principal")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			mbIDStr, err := request.RequireString("master_bill_id")
			if err != nil {
				return mcp.NewToolResultError("master_bill_id es requerido"), nil
			}
			mbID, err := uuid.Parse(mbIDStr)
			if err != nil {
				return mcp.NewToolResultError("UUID master_bill_id inválido"), nil
			}

			bills, err := params.BillingService.CalculateMasterBill(ctx, mbID)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			data, _ := json.MarshalIndent(bills, "", "  ")
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// Tool 6: get_tenant_bill
	s.AddTool(
		mcp.NewTool(
			"get_tenant_bill",
			mcp.WithDescription("Obtiene el recibo de un inquilino con su desglose transparente: consumo individual medido, cuota por cada cargo extra, y alerta de sobreconsumo si aplica."),
			mcp.WithString("tenant_bill_id", mcp.Required(), mcp.Description("UUID del recibo del inquilino")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			idStr, err := request.RequireString("tenant_bill_id")
			if err != nil {
				return mcp.NewToolResultError("tenant_bill_id es requerido"), nil
			}
			id, err := uuid.Parse(idStr)
			if err != nil {
				return mcp.NewToolResultError("UUID tenant_bill_id inválido"), nil
			}

			bill, err := params.BillingService.GetTenantBill(ctx, id)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			data, _ := json.MarshalIndent(bill, "", "  ")
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// Tool 7: get_tenant_history
	s.AddTool(
		mcp.NewTool(
			"get_tenant_history",
			mcp.WithDescription("Consulta el histórico de recibos y tendencias de consumo de un inquilino."),
			mcp.WithString("tenant_id", mcp.Required(), mcp.Description("UUID del inquilino")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			idStr, err := request.RequireString("tenant_id")
			if err != nil {
				return mcp.NewToolResultError("tenant_id es requerido"), nil
			}
			id, err := uuid.Parse(idStr)
			if err != nil {
				return mcp.NewToolResultError("UUID tenant_id inválido"), nil
			}

			bills, err := params.BillingService.ListTenantBillsByTenant(ctx, id)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			data, _ := json.MarshalIndent(bills, "", "  ")
			return mcp.NewToolResultText(fmt.Sprintf("Historial de %d recibos:\n%s", len(bills), string(data))), nil
		},
	)
}
