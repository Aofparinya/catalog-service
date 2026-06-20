CREATE SEQUENCE IF NOT EXISTS catalog.product_number_seq START WITH 1 INCREMENT BY 1;

CREATE TABLE catalog.categories (
    id UUID PRIMARY KEY,
    code VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    parent_id UUID REFERENCES catalog.categories(id),
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT categories_status_check CHECK (status IN ('ACTIVE', 'INACTIVE')),
    CONSTRAINT categories_parent_check CHECK (parent_id IS NULL OR parent_id <> id)
);
CREATE UNIQUE INDEX categories_code_unique ON catalog.categories(lower(code)) WHERE deleted_at IS NULL;
CREATE INDEX categories_parent_idx ON catalog.categories(parent_id) WHERE deleted_at IS NULL;

CREATE TABLE catalog.products (
    id UUID PRIMARY KEY,
    product_no VARCHAR(32) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    created_by UUID,
    updated_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT products_status_check CHECK (status IN ('DRAFT', 'ACTIVE', 'INACTIVE'))
);
CREATE INDEX products_search_idx ON catalog.products(lower(name), product_no) WHERE deleted_at IS NULL;

CREATE TABLE catalog.product_categories (
    product_id UUID NOT NULL REFERENCES catalog.products(id),
    category_id UUID NOT NULL REFERENCES catalog.categories(id),
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY(product_id, category_id)
);
CREATE UNIQUE INDEX product_categories_primary_unique
    ON catalog.product_categories(product_id) WHERE is_primary = TRUE;

CREATE TABLE catalog.product_images (
    id UUID PRIMARY KEY,
    product_id UUID NOT NULL REFERENCES catalog.products(id),
    file_id UUID NOT NULL,
    alt_text VARCHAR(255),
    sort_order INT NOT NULL DEFAULT 0,
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX product_images_primary_unique
    ON catalog.product_images(product_id) WHERE is_primary = TRUE AND deleted_at IS NULL;

CREATE TABLE catalog.skus (
    id UUID PRIMARY KEY,
    product_id UUID NOT NULL REFERENCES catalog.products(id),
    code VARCHAR(100) NOT NULL,
    barcode VARCHAR(100),
    name VARCHAR(255) NOT NULL,
    attributes JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT skus_status_check CHECK (status IN ('ACTIVE', 'INACTIVE')),
    CONSTRAINT skus_attributes_object CHECK (jsonb_typeof(attributes) = 'object')
);
CREATE UNIQUE INDEX skus_code_unique ON catalog.skus(lower(code)) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX skus_barcode_unique ON catalog.skus(lower(barcode)) WHERE barcode IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX skus_product_idx ON catalog.skus(product_id) WHERE deleted_at IS NULL;

CREATE TABLE catalog.prices (
    id UUID PRIMARY KEY,
    sku_id UUID NOT NULL REFERENCES catalog.skus(id),
    amount NUMERIC(18,2) NOT NULL,
    currency CHAR(3) NOT NULL,
    valid_from TIMESTAMPTZ NOT NULL,
    valid_to TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT prices_amount_check CHECK (amount >= 0),
    CONSTRAINT prices_period_check CHECK (valid_to IS NULL OR valid_to > valid_from)
);
CREATE INDEX prices_sku_currency_idx ON catalog.prices(sku_id, currency, valid_from) WHERE deleted_at IS NULL;

CREATE TABLE catalog.warehouses (
    id UUID PRIMARY KEY,
    code VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT warehouses_status_check CHECK (status IN ('ACTIVE', 'INACTIVE'))
);
CREATE UNIQUE INDEX warehouses_code_unique ON catalog.warehouses(lower(code)) WHERE deleted_at IS NULL;

CREATE TABLE catalog.inventory_stocks (
    warehouse_id UUID NOT NULL REFERENCES catalog.warehouses(id),
    sku_id UUID NOT NULL REFERENCES catalog.skus(id),
    on_hand BIGINT NOT NULL DEFAULT 0,
    reserved BIGINT NOT NULL DEFAULT 0,
    reorder_level BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY(warehouse_id, sku_id),
    CONSTRAINT inventory_stock_nonnegative CHECK (on_hand >= 0 AND reserved >= 0 AND reserved <= on_hand),
    CONSTRAINT inventory_reorder_nonnegative CHECK (reorder_level >= 0)
);

CREATE TABLE catalog.stock_movements (
    id UUID PRIMARY KEY,
    warehouse_id UUID NOT NULL REFERENCES catalog.warehouses(id),
    sku_id UUID NOT NULL REFERENCES catalog.skus(id),
    movement_type VARCHAR(30) NOT NULL,
    on_hand_change BIGINT NOT NULL DEFAULT 0,
    reserved_change BIGINT NOT NULL DEFAULT 0,
    on_hand_after BIGINT NOT NULL,
    reserved_after BIGINT NOT NULL,
    reference_type VARCHAR(100),
    reference_id VARCHAR(150),
    note TEXT,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT stock_movements_type_check CHECK (
        movement_type IN ('ADJUSTMENT', 'RESERVE', 'CONFIRM', 'RELEASE', 'EXPIRE')
    )
);
CREATE INDEX stock_movements_lookup_idx
    ON catalog.stock_movements(warehouse_id, sku_id, created_at DESC);

CREATE TABLE catalog.stock_reservations (
    id UUID PRIMARY KEY,
    warehouse_id UUID NOT NULL REFERENCES catalog.warehouses(id),
    reference_type VARCHAR(100) NOT NULL,
    reference_id VARCHAR(150) NOT NULL,
    request_hash CHAR(64) NOT NULL,
    status VARCHAR(20) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    confirmed_at TIMESTAMPTZ,
    released_at TIMESTAMPTZ,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT reservations_status_check CHECK (status IN ('PENDING', 'CONFIRMED', 'RELEASED', 'EXPIRED'))
);
CREATE UNIQUE INDEX stock_reservations_reference_unique
    ON catalog.stock_reservations(reference_type, reference_id);
CREATE INDEX stock_reservations_expiry_idx
    ON catalog.stock_reservations(status, expires_at) WHERE status = 'PENDING';

CREATE TABLE catalog.stock_reservation_items (
    reservation_id UUID NOT NULL REFERENCES catalog.stock_reservations(id),
    sku_id UUID NOT NULL REFERENCES catalog.skus(id),
    quantity BIGINT NOT NULL,
    PRIMARY KEY(reservation_id, sku_id),
    CONSTRAINT reservation_items_quantity_check CHECK (quantity > 0)
);
