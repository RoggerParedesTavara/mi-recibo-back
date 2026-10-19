# Mi Recibo - Backend (Go + PostgreSQL 16)

Backend para la gestión y prorrateo automatizado de recibos de servicios básicos (luz, agua, gas) en inmuebles de alquiler con múltiples inquilinos.

---

## 🚀 Características Principales

1. **Gestión de Propietarios, Propiedades e Inquilinos**:
   - Un dueño puede administrar múltiples casas o departamentos.
   - Cada casa alberga a múltiples inquilinos (activos/inactivos).

2. **Lista Dinámica de Cargos Extras e Impuestos**:
   - Cada recibo principal admite una lista arbitraria de cargos adicionales (ej. *Mantenimiento de red*, *Alumbrado público*, *Cargo fijo*, etc.).
   - Cada ítem contiene su propia descripción y monto.
   - Estos cargos se dividen en partes iguales entre la cantidad $N$ de inquilinos activos de la propiedad (sin incluir al dueño).
   - Ejemplo: Cargo de mantenimiento de \$300 entre 3 inquilinos = \$100 a cada uno.

3. **Consumo Propio Estricto (Sin redistribución)**:
   - Cada inquilino paga **únicamente y de forma estricta su consumo medido**:
     $$\text{Costo Consumo} = \text{unidades consumidas} \times \text{precio unitario}$$
   - Las diferencias entre el recibo principal y las lecturas individuales (áreas comunes o pérdidas) no se distribuyen a los inquilinos.

4. **Detección y Alertas de Sobreconsumo**:
   - Compara las unidades consumidas del mes contra el mes anterior del mismo servicio.
   - Si el incremento supera el umbral configurado (por defecto 20%), marca el recibo con `alert_spike = true` y detalla la variación porcentual.

5. **Desglose Transparente (Breakdown)**:
   - Cada inquilino puede consultar su recibo con el desglose exacto de cada cargo extra: concepto original, costo total facturado en la propiedad, cantidad de inquilinos ($N$), y la cuota individual que le corresponde.

6. **Servidor MCP (Model Context Protocol)**:
   - Incluye binario en `cmd/mcp` para que asistentes de IA (Claude Desktop, Antigravity, Cursor) puedan consultar consumos, reportes y registrar lecturas por lenguaje natural.

---

## 🛠️ Stack Tecnológico

- **Lenguaje**: Go 1.24+ / Go 1.27
- **Base de Datos**: PostgreSQL 16+
- **Driver de BD**: `pgx/v5` (`pgxpool`) con transacciones ACID (`pgx.Tx`)
- **Enrutador HTTP**: `go-chi/chi/v5`
- **Aritmética Decimal**: `shopspring/decimal` (precisión financiera exacta sin errores de punto flotante)
- **Protocolo de IA**: `mark3labs/mcp-go` (JSON-RPC stdio)

---

## 📁 Estructura del Proyecto

```
mi-recibo/
├── cmd/
│   ├── api/
│   │   ├── main.go                  # Entrypoint del servidor HTTP REST
│   │   └── e2e_test.go              # Test de integración end-to-end
│   └── mcp/
│       └── main.go                  # Entrypoint del servidor MCP (stdio)
├── internal/
│   ├── config/                      # Configuración y variables de entorno
│   ├── domain/                      # Modelos de dominio y motor de cálculo puro
│   │   ├── owner.go
│   │   ├── property.go
│   │   ├── tenant.go
│   │   ├── master_bill.go           # Recibo principal y lista dinámica de cargos
│   │   ├── tenant_bill.go           # Recibo individual y desglose detallado
│   │   ├── consumption.go           # Lecturas y alertas de sobreconsumo
│   │   ├── calculator.go            # Fórmulas de prorrateo
│   │   └── calculator_test.go       # Tests unitarios del cálculo
│   ├── repository/postgres/         # Capa de persistencia con pgxpool
│   │   ├── db.go                    # Conexión y runner de migraciones
│   │   ├── owner_repo.go
│   │   ├── property_repo.go
│   │   ├── tenant_repo.go
│   │   ├── master_bill_repo.go
│   │   ├── consumption_repo.go
│   │   └── tenant_bill_repo.go
│   ├── service/                     # Casos de uso de negocio
│   │   ├── property_service.go
│   │   ├── tenant_service.go
│   │   └── billing_service.go
│   └── handler/
│       ├── http/                    # Controladores y rutas REST
│       │   ├── router.go
│       │   ├── owner_handler.go
│       │   ├── property_handler.go
│       │   ├── tenant_handler.go
│       │   ├── bill_handler.go
│       │   └── response.go
│       └── mcp/                     # Herramientas expuestas para agentes de IA
│           └── tools.go
├── docs/                            # Documentación técnica y especificaciones
│   ├── openapi.yaml                 # Especificación OpenAPI 3.0 / Swagger
│   ├── architecture.md              # Documento de arquitectura y decisiones de diseño
│   └── mi-recibo.postman_collection.json # Colección de Postman / Bruno
├── migrations/                      # Migraciones SQL versionadas
│   ├── 000001_init_schema.up.sql
│   └── 000001_init_schema.down.sql
├── docker-compose.yml               # Entorno PostgreSQL 16
├── Dockerfile                       # Multi-stage build optimizado
├── go.mod
└── go.sum
```

---

## 📚 Documentación Técnica Adicional

- 📄 **[Especificación OpenAPI 3.0 (Swagger)](docs/openapi.yaml)**: Contrato OpenAPI 3.0.3 completo para importar en Postman, Insomnia, Swagger UI o Bruno.
- 📐 **[Documento de Arquitectura y Diseño](docs/architecture.md)**: Explicación de capas Clean Architecture, diagramas ER, reglas matemáticas de prorrateo, aislamiento de consumo y servidor MCP.
- 📬 **[Colección de Postman](docs/mi-recibo.postman_collection.json)**: Colección preconfigurada con variables para ejecutar y probar todas las llamadas de la API.

---

## ⚡ Inicio Rápido

### 1. Requisitos
- Go 1.22 o superior
- PostgreSQL 16 (vía Docker o instalación local)

### 2. Configurar variables de entorno
Copia el archivo de ejemplo:
```bash
cp .env.example .env
```

### 3. Levantar Base de Datos

**Opción A - Con Docker Compose:**
```bash
docker compose up -d
```

**Opción B - Con PostgreSQL local (Homebrew en macOS):**
```bash
brew services start postgresql@16
createdb mi_recibo
```

### 4. Compilar y Ejecutar la API
```bash
go run cmd/api/main.go
```
*Las migraciones SQL se aplican automáticamente al iniciar.*

La API estará disponible en `http://localhost:8080`.

---

## 🧪 Ejecutar Tests

```bash
# Tests unitarios del motor de cálculo
go test -v ./internal/domain/...

# Tests de integración end-to-end con PostgreSQL
go test -v ./cmd/api/...

# Todos los tests
go test -v ./...
```

---

## 📖 Guía de la API REST

### 1. Registrar Dueño
```bash
curl -X POST http://localhost:8080/api/v1/owners \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Carlos Dueño",
    "email": "carlos@ejemplo.com",
    "phone": "+54911223344"
  }'
```

### 2. Registrar Propiedad (Casa)
```bash
curl -X POST http://localhost:8080/api/v1/owners/{owner_id}/properties \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Casa Los Rosales",
    "address": "Av. Principal 123"
  }'
```

### 3. Registrar Inquilinos
```bash
curl -X POST http://localhost:8080/api/v1/properties/{property_id}/tenants \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Juan Pérez",
    "email": "juan@ejemplo.com",
    "phone": "+54911556677",
    "room_or_unit": "Habitación 1"
  }'
```

### 4. Cargar Recibo Principal con Lista Dinámica de Cargos Extras
```bash
curl -X POST http://localhost:8080/api/v1/properties/{property_id}/master-bills \
  -H "Content-Type: application/json" \
  -d '{
    "service_type": "LUZ",
    "billing_year": 2026,
    "billing_month": 10,
    "total_consumption": 350.0,
    "unit_price": 0.50,
    "extra_charges": [
      {
        "description": "Mantenimiento de la red",
        "amount": 300.00
      },
      {
        "description": "Alumbrado público",
        "amount": 45.00
      },
      {
        "description": "Cargo fijo mensual",
        "amount": 12.00
      }
    ]
  }'
```

### 5. Cargar Consumos Mensuales de Inquilinos
```bash
curl -X POST http://localhost:8080/api/v1/master-bills/{master_bill_id}/consumptions \
  -H "Content-Type: application/json" \
  -d '{
    "readings": [
      {
        "tenant_id": "{tenant_1_id}",
        "consumed_units": 80.0
      },
      {
        "tenant_id": "{tenant_2_id}",
        "consumed_units": 120.0
      },
      {
        "tenant_id": "{tenant_3_id}",
        "consumed_units": 100.0
      }
    ]
  }'
```

### 6. Ejecutar Liquidación y Cálculo Automático
```bash
curl -X POST http://localhost:8080/api/v1/master-bills/{master_bill_id}/calculate
```

### 7. Consultar Recibo Individual del Inquilino con Desglose
```bash
curl -X GET http://localhost:8080/api/v1/tenant-bills/{tenant_bill_id}
```
*Ejemplo de Respuesta:*
```json
{
  "id": "e5f6a1b2-...",
  "tenant_id": "a1b2c3d4-...",
  "tenant_name": "Juan Pérez",
  "room_or_unit": "Habitación 1",
  "property_name": "Casa Los Rosales",
  "service_type": "LUZ",
  "billing_year": 2026,
  "billing_month": 10,
  "consumed_units": "80",
  "unit_price": "0.5",
  "consumption_amount": "40",
  "extra_charges_total_share": "119",
  "total_amount": "159",
  "status": "PENDING",
  "alert_spike": true,
  "alert_message": "El consumo aumentó un 33.33% respecto al mes anterior (de 60.00 a 80.00 unidades).",
  "previous_month_units": "60",
  "breakdowns": [
    {
      "description": "Mantenimiento de la red",
      "original_amount": "300",
      "tenants_count": 3,
      "tenant_share": "100"
    },
    {
      "description": "Alumbrado público",
      "original_amount": "45",
      "tenants_count": 3,
      "tenant_share": "15"
    },
    {
      "description": "Cargo fijo mensual",
      "original_amount": "12",
      "tenants_count": 3,
      "tenant_share": "4"
    }
  ]
}
```

### 8. Marcar Recibo como Pagado
```bash
curl -X PATCH http://localhost:8080/api/v1/tenant-bills/{tenant_bill_id}/payment \
  -H "Content-Type: application/json" \
  -d '{"is_paid": true}'
```

---

## 🤖 Uso del Servidor MCP para Asistentes de IA

Para conectar asistentes de IA (Claude Desktop, Antigravity, Cursor) al backend:

### Configuración en `claude_desktop_config.json` o settings de Antigravity:
```json
{
  "mcpServers": {
    "mi-recibo": {
      "command": "/ruta/a/mi-recibo/mcp",
      "env": {
        "DB_HOST": "localhost",
        "DB_PORT": "5432",
        "DB_USER": "postgres",
        "DB_PASSWORD": "postgres",
        "DB_NAME": "mi_recibo",
        "DB_SSLMODE": "disable"
      }
    }
  }
}
```

### Herramientas MCP disponibles:
- `list_properties`: Lista las propiedades de un dueño.
- `list_tenants`: Lista los inquilinos de una casa.
- `get_master_bill`: Consulta los datos del recibo principal y sus cargos extras.
- `record_tenant_reading`: Registra la lectura mensual de un inquilino y evalúa incrementos inusuales.
- `calculate_master_bill`: Liquida el recibo del mes y genera los recibos individuales.
- `get_tenant_bill`: Consulta un recibo de inquilino con desglose transparente.
- `get_tenant_history`: Consulta el historial de consumos y recibos de un inquilino.
