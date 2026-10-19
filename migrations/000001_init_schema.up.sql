-- Enable UUID extension if needed
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 1. Owners
CREATE TABLE IF NOT EXISTS owners (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    phone VARCHAR(50),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 2. Properties
CREATE TABLE IF NOT EXISTS properties (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES owners(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    address TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 3. Tenants
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    property_id UUID NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    phone VARCHAR(50),
    room_or_unit VARCHAR(100),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 4. Master Bills
CREATE TABLE IF NOT EXISTS master_bills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    property_id UUID NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    service_type VARCHAR(50) NOT NULL, -- 'LUZ', 'AGUA', 'GAS'
    billing_year INT NOT NULL,
    billing_month INT NOT NULL,
    total_consumption NUMERIC(14, 4) NOT NULL DEFAULT 0,
    unit_price NUMERIC(14, 4) NOT NULL DEFAULT 0,
    consumption_amount NUMERIC(12, 2) NOT NULL DEFAULT 0,
    extra_charges_total NUMERIC(12, 2) NOT NULL DEFAULT 0,
    total_amount NUMERIC(12, 2) NOT NULL DEFAULT 0,
    issue_date DATE,
    due_date DATE,
    status VARCHAR(50) NOT NULL DEFAULT 'DRAFT', -- 'DRAFT', 'CALCULATED', 'PAID'
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_property_service_period UNIQUE (property_id, service_type, billing_year, billing_month)
);

-- 5. Master Bill Extra Charges (Dynamic List with description and price)
CREATE TABLE IF NOT EXISTS master_bill_extra_charges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    master_bill_id UUID NOT NULL REFERENCES master_bills(id) ON DELETE CASCADE,
    description VARCHAR(255) NOT NULL,
    amount NUMERIC(12, 2) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 6. Tenant Monthly Consumptions (Uploaded by owner)
CREATE TABLE IF NOT EXISTS tenant_consumptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    master_bill_id UUID NOT NULL REFERENCES master_bills(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    consumed_units NUMERIC(14, 4) NOT NULL,
    previous_reading NUMERIC(14, 4),
    current_reading NUMERIC(14, 4),
    previous_month_consumption NUMERIC(14, 4),
    variation_percentage NUMERIC(8, 2),
    alert_spike BOOLEAN NOT NULL DEFAULT false,
    alert_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_master_bill_tenant UNIQUE (master_bill_id, tenant_id)
);

-- 7. Tenant Bills (Calculated individual receipts)
CREATE TABLE IF NOT EXISTS tenant_bills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    master_bill_id UUID NOT NULL REFERENCES master_bills(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    consumed_units NUMERIC(14, 4) NOT NULL,
    unit_price NUMERIC(14, 4) NOT NULL,
    consumption_amount NUMERIC(12, 2) NOT NULL,
    extra_charges_total_share NUMERIC(12, 2) NOT NULL DEFAULT 0,
    total_amount NUMERIC(12, 2) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING', -- 'PENDING', 'PAID'
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_tenant_bill_master_tenant UNIQUE (master_bill_id, tenant_id)
);

-- 8. Tenant Bill Charge Breakdowns (Itemized extra charges breakdown for each tenant)
CREATE TABLE IF NOT EXISTS tenant_bill_charge_breakdowns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_bill_id UUID NOT NULL REFERENCES tenant_bills(id) ON DELETE CASCADE,
    extra_charge_id UUID REFERENCES master_bill_extra_charges(id) ON DELETE SET NULL,
    description VARCHAR(255) NOT NULL,
    original_amount NUMERIC(12, 2) NOT NULL,
    tenants_count INT NOT NULL,
    tenant_share NUMERIC(12, 2) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indices for performance
CREATE INDEX IF NOT EXISTS idx_properties_owner_id ON properties(owner_id);
CREATE INDEX IF NOT EXISTS idx_tenants_property_id ON tenants(property_id);
CREATE INDEX IF NOT EXISTS idx_master_bills_property_id ON master_bills(property_id);
CREATE INDEX IF NOT EXISTS idx_master_bill_charges_mb_id ON master_bill_extra_charges(master_bill_id);
CREATE INDEX IF NOT EXISTS idx_tenant_consumptions_mb_id ON tenant_consumptions(master_bill_id);
CREATE INDEX IF NOT EXISTS idx_tenant_consumptions_tenant_id ON tenant_consumptions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_bills_mb_id ON tenant_bills(master_bill_id);
CREATE INDEX IF NOT EXISTS idx_tenant_bills_tenant_id ON tenant_bills(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_bill_breakdowns_tb_id ON tenant_bill_charge_breakdowns(tenant_bill_id);
