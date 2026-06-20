package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Status string
type ProductStatus string
type ReservationStatus string

const (
	StatusActive   Status = "ACTIVE"
	StatusInactive Status = "INACTIVE"

	ProductStatusDraft    ProductStatus = "DRAFT"
	ProductStatusActive   ProductStatus = "ACTIVE"
	ProductStatusInactive ProductStatus = "INACTIVE"

	ReservationPending   ReservationStatus = "PENDING"
	ReservationConfirmed ReservationStatus = "CONFIRMED"
	ReservationReleased  ReservationStatus = "RELEASED"
	ReservationExpired   ReservationStatus = "EXPIRED"
)

type Category struct {
	ID          uuid.UUID  `json:"id"`
	Code        string     `json:"code"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	ParentID    *uuid.UUID `json:"parentId,omitempty"`
	Status      Status     `json:"status"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type Product struct {
	ID          uuid.UUID     `json:"id"`
	ProductNo   string        `json:"productNo"`
	Name        string        `json:"name"`
	Description *string       `json:"description,omitempty"`
	Status      ProductStatus `json:"status"`
	CreatedBy   *uuid.UUID    `json:"createdBy,omitempty"`
	UpdatedBy   *uuid.UUID    `json:"updatedBy,omitempty"`
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt"`
}

type ProductCategory struct {
	ProductID  uuid.UUID `json:"productId"`
	CategoryID uuid.UUID `json:"categoryId"`
	IsPrimary  bool      `json:"isPrimary"`
	CreatedAt  time.Time `json:"createdAt"`
}

type ProductImage struct {
	ID        uuid.UUID `json:"id"`
	ProductID uuid.UUID `json:"productId"`
	FileID    uuid.UUID `json:"fileId"`
	AltText   *string   `json:"altText,omitempty"`
	SortOrder int       `json:"sortOrder"`
	IsPrimary bool      `json:"isPrimary"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type SKU struct {
	ID         uuid.UUID       `json:"id"`
	ProductID  uuid.UUID       `json:"productId"`
	Code       string          `json:"code"`
	Barcode    *string         `json:"barcode,omitempty"`
	Name       string          `json:"name"`
	Attributes json.RawMessage `json:"attributes"`
	Status     Status          `json:"status"`
	CreatedAt  time.Time       `json:"createdAt"`
	UpdatedAt  time.Time       `json:"updatedAt"`
}

type Price struct {
	ID        uuid.UUID       `json:"id"`
	SKUID     uuid.UUID       `json:"skuId"`
	Amount    decimal.Decimal `json:"amount"`
	Currency  string          `json:"currency"`
	ValidFrom time.Time       `json:"validFrom"`
	ValidTo   *time.Time      `json:"validTo,omitempty"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

type Warehouse struct {
	ID        uuid.UUID `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Status    Status    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Stock struct {
	WarehouseID  uuid.UUID `json:"warehouseId"`
	SKUID        uuid.UUID `json:"skuId"`
	OnHand       int64     `json:"onHand"`
	Reserved     int64     `json:"reserved"`
	Available    int64     `json:"available"`
	ReorderLevel int64     `json:"reorderLevel"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type StockMovement struct {
	ID             uuid.UUID  `json:"id"`
	WarehouseID    uuid.UUID  `json:"warehouseId"`
	SKUID          uuid.UUID  `json:"skuId"`
	MovementType   string     `json:"movementType"`
	OnHandChange   int64      `json:"onHandChange"`
	ReservedChange int64      `json:"reservedChange"`
	OnHandAfter    int64      `json:"onHandAfter"`
	ReservedAfter  int64      `json:"reservedAfter"`
	ReferenceType  *string    `json:"referenceType,omitempty"`
	ReferenceID    *string    `json:"referenceId,omitempty"`
	Note           *string    `json:"note,omitempty"`
	CreatedBy      *uuid.UUID `json:"createdBy,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
}

type Reservation struct {
	ID            uuid.UUID         `json:"id"`
	WarehouseID   uuid.UUID         `json:"warehouseId"`
	ReferenceType string            `json:"referenceType"`
	ReferenceID   string            `json:"referenceId"`
	Status        ReservationStatus `json:"status"`
	ExpiresAt     time.Time         `json:"expiresAt"`
	ConfirmedAt   *time.Time        `json:"confirmedAt,omitempty"`
	ReleasedAt    *time.Time        `json:"releasedAt,omitempty"`
	Items         []ReservationItem `json:"items"`
	CreatedAt     time.Time         `json:"createdAt"`
	UpdatedAt     time.Time         `json:"updatedAt"`
}

type ReservationItem struct {
	SKUID    uuid.UUID `json:"skuId"`
	Quantity int64     `json:"quantity"`
}
