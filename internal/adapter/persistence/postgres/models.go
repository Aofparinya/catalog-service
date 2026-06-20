package postgres

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type categoryModel struct {
	ID          uuid.UUID      `gorm:"column:id;type:uuid;primaryKey"`
	Code        string         `gorm:"column:code"`
	Name        string         `gorm:"column:name"`
	Description *string        `gorm:"column:description"`
	ParentID    *uuid.UUID     `gorm:"column:parent_id;type:uuid"`
	Status      string         `gorm:"column:status"`
	CreatedAt   time.Time      `gorm:"column:created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at"`
}

func (categoryModel) TableName() string { return "catalog.categories" }

type productModel struct {
	ID          uuid.UUID      `gorm:"column:id;type:uuid;primaryKey"`
	ProductNo   string         `gorm:"column:product_no"`
	Name        string         `gorm:"column:name"`
	Description *string        `gorm:"column:description"`
	Status      string         `gorm:"column:status"`
	CreatedBy   *uuid.UUID     `gorm:"column:created_by;type:uuid"`
	UpdatedBy   *uuid.UUID     `gorm:"column:updated_by;type:uuid"`
	CreatedAt   time.Time      `gorm:"column:created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at"`
}

func (productModel) TableName() string { return "catalog.products" }

type productCategoryModel struct {
	ProductID  uuid.UUID `gorm:"column:product_id;type:uuid;primaryKey"`
	CategoryID uuid.UUID `gorm:"column:category_id;type:uuid;primaryKey"`
	IsPrimary  bool      `gorm:"column:is_primary"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

func (productCategoryModel) TableName() string { return "catalog.product_categories" }

type productImageModel struct {
	ID        uuid.UUID      `gorm:"column:id;type:uuid;primaryKey"`
	ProductID uuid.UUID      `gorm:"column:product_id;type:uuid"`
	FileID    uuid.UUID      `gorm:"column:file_id;type:uuid"`
	AltText   *string        `gorm:"column:alt_text"`
	SortOrder int            `gorm:"column:sort_order"`
	IsPrimary bool           `gorm:"column:is_primary"`
	CreatedAt time.Time      `gorm:"column:created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at"`
}

func (productImageModel) TableName() string { return "catalog.product_images" }

type skuModel struct {
	ID         uuid.UUID       `gorm:"column:id;type:uuid;primaryKey"`
	ProductID  uuid.UUID       `gorm:"column:product_id;type:uuid"`
	Code       string          `gorm:"column:code"`
	Barcode    *string         `gorm:"column:barcode"`
	Name       string          `gorm:"column:name"`
	Attributes json.RawMessage `gorm:"column:attributes;type:jsonb"`
	Status     string          `gorm:"column:status"`
	CreatedAt  time.Time       `gorm:"column:created_at"`
	UpdatedAt  time.Time       `gorm:"column:updated_at"`
	DeletedAt  gorm.DeletedAt  `gorm:"column:deleted_at"`
}

func (skuModel) TableName() string { return "catalog.skus" }

type priceModel struct {
	ID        uuid.UUID       `gorm:"column:id;type:uuid;primaryKey"`
	SKUID     uuid.UUID       `gorm:"column:sku_id;type:uuid"`
	Amount    decimal.Decimal `gorm:"column:amount;type:numeric(18,2)"`
	Currency  string          `gorm:"column:currency"`
	ValidFrom time.Time       `gorm:"column:valid_from"`
	ValidTo   *time.Time      `gorm:"column:valid_to"`
	CreatedAt time.Time       `gorm:"column:created_at"`
	UpdatedAt time.Time       `gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt  `gorm:"column:deleted_at"`
}

func (priceModel) TableName() string { return "catalog.prices" }

type warehouseModel struct {
	ID        uuid.UUID      `gorm:"column:id;type:uuid;primaryKey"`
	Code      string         `gorm:"column:code"`
	Name      string         `gorm:"column:name"`
	Status    string         `gorm:"column:status"`
	CreatedAt time.Time      `gorm:"column:created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at"`
}

func (warehouseModel) TableName() string { return "catalog.warehouses" }

type stockModel struct {
	WarehouseID  uuid.UUID `gorm:"column:warehouse_id;type:uuid;primaryKey"`
	SKUID        uuid.UUID `gorm:"column:sku_id;type:uuid;primaryKey"`
	OnHand       int64     `gorm:"column:on_hand"`
	Reserved     int64     `gorm:"column:reserved"`
	ReorderLevel int64     `gorm:"column:reorder_level"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (stockModel) TableName() string { return "catalog.inventory_stocks" }

type movementModel struct {
	ID             uuid.UUID  `gorm:"column:id;type:uuid;primaryKey"`
	WarehouseID    uuid.UUID  `gorm:"column:warehouse_id;type:uuid"`
	SKUID          uuid.UUID  `gorm:"column:sku_id;type:uuid"`
	MovementType   string     `gorm:"column:movement_type"`
	OnHandChange   int64      `gorm:"column:on_hand_change"`
	ReservedChange int64      `gorm:"column:reserved_change"`
	OnHandAfter    int64      `gorm:"column:on_hand_after"`
	ReservedAfter  int64      `gorm:"column:reserved_after"`
	ReferenceType  *string    `gorm:"column:reference_type"`
	ReferenceID    *string    `gorm:"column:reference_id"`
	Note           *string    `gorm:"column:note"`
	CreatedBy      *uuid.UUID `gorm:"column:created_by;type:uuid"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
}

func (movementModel) TableName() string { return "catalog.stock_movements" }

type reservationModel struct {
	ID            uuid.UUID  `gorm:"column:id;type:uuid;primaryKey"`
	WarehouseID   uuid.UUID  `gorm:"column:warehouse_id;type:uuid"`
	ReferenceType string     `gorm:"column:reference_type"`
	ReferenceID   string     `gorm:"column:reference_id"`
	RequestHash   string     `gorm:"column:request_hash"`
	Status        string     `gorm:"column:status"`
	ExpiresAt     time.Time  `gorm:"column:expires_at"`
	ConfirmedAt   *time.Time `gorm:"column:confirmed_at"`
	ReleasedAt    *time.Time `gorm:"column:released_at"`
	CreatedBy     *uuid.UUID `gorm:"column:created_by;type:uuid"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
}

func (reservationModel) TableName() string { return "catalog.stock_reservations" }

type reservationItemModel struct {
	ReservationID uuid.UUID `gorm:"column:reservation_id;type:uuid;primaryKey"`
	SKUID         uuid.UUID `gorm:"column:sku_id;type:uuid;primaryKey"`
	Quantity      int64     `gorm:"column:quantity"`
}

func (reservationItemModel) TableName() string { return "catalog.stock_reservation_items" }
