package application

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/saaof/order-platform/catalog-service/internal/domain"
	"github.com/shopspring/decimal"
)

type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"totalPages"`
}

type Page[T any] struct {
	Data       []T        `json:"data"`
	Pagination Pagination `json:"pagination"`
}

type ListFilter struct {
	Query         string
	Status        string
	CategoryID    *uuid.UUID
	WarehouseID   *uuid.UUID
	SKUID         *uuid.UUID
	LowStock      bool
	ReferenceType string
	ReferenceID   string
	Page          int
	PageSize      int
}

type CreateCategoryInput struct {
	Code        string        `json:"code" validate:"required,max=100"`
	Name        string        `json:"name" validate:"required,max=255"`
	Description *string       `json:"description"`
	ParentID    *uuid.UUID    `json:"parentId"`
	Status      domain.Status `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
}
type UpdateCategoryInput struct {
	Code        *string        `json:"code" validate:"omitempty,max=100"`
	Name        *string        `json:"name" validate:"omitempty,max=255"`
	Description *string        `json:"description"`
	ParentID    *uuid.UUID     `json:"parentId"`
	Status      *domain.Status `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
}

type CreateProductInput struct {
	Name        string               `json:"name" validate:"required,max=255"`
	Description *string              `json:"description"`
	Status      domain.ProductStatus `json:"status" validate:"omitempty,oneof=DRAFT ACTIVE INACTIVE"`
}
type UpdateProductInput struct {
	Name        *string               `json:"name" validate:"omitempty,max=255"`
	Description *string               `json:"description"`
	Status      *domain.ProductStatus `json:"status" validate:"omitempty,oneof=DRAFT ACTIVE INACTIVE"`
}
type AssignCategoryInput struct {
	CategoryID uuid.UUID `json:"categoryId" validate:"required"`
	IsPrimary  bool      `json:"isPrimary"`
}
type CreateImageInput struct {
	FileID    uuid.UUID `json:"fileId" validate:"required"`
	AltText   *string   `json:"altText"`
	SortOrder int       `json:"sortOrder" validate:"min=0"`
	IsPrimary bool      `json:"isPrimary"`
}
type UpdateImageInput struct {
	AltText   *string `json:"altText"`
	SortOrder *int    `json:"sortOrder" validate:"omitempty,min=0"`
	IsPrimary *bool   `json:"isPrimary"`
}

type CreateSKUInput struct {
	Code       string          `json:"code" validate:"required,max=100"`
	Barcode    *string         `json:"barcode" validate:"omitempty,max=100"`
	Name       string          `json:"name" validate:"required,max=255"`
	Attributes json.RawMessage `json:"attributes"`
	Status     domain.Status   `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
}
type UpdateSKUInput struct {
	Code       *string          `json:"code" validate:"omitempty,max=100"`
	Barcode    *string          `json:"barcode" validate:"omitempty,max=100"`
	Name       *string          `json:"name" validate:"omitempty,max=255"`
	Attributes *json.RawMessage `json:"attributes"`
	Status     *domain.Status   `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
}

type CreatePriceInput struct {
	Amount    decimal.Decimal `json:"amount" validate:"required"`
	Currency  string          `json:"currency" validate:"required,len=3"`
	ValidFrom time.Time       `json:"validFrom" validate:"required"`
	ValidTo   *time.Time      `json:"validTo"`
}
type UpdatePriceInput struct {
	Amount    *decimal.Decimal `json:"amount"`
	Currency  *string          `json:"currency" validate:"omitempty,len=3"`
	ValidFrom *time.Time       `json:"validFrom"`
	ValidTo   *time.Time       `json:"validTo"`
}

type CreateWarehouseInput struct {
	Code   string        `json:"code" validate:"required,max=100"`
	Name   string        `json:"name" validate:"required,max=255"`
	Status domain.Status `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
}
type UpdateWarehouseInput struct {
	Code   *string        `json:"code" validate:"omitempty,max=100"`
	Name   *string        `json:"name" validate:"omitempty,max=255"`
	Status *domain.Status `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
}

type AdjustmentInput struct {
	WarehouseID   uuid.UUID `json:"warehouseId" validate:"required"`
	SKUID         uuid.UUID `json:"skuId" validate:"required"`
	Quantity      int64     `json:"quantity" validate:"required"`
	ReorderLevel  *int64    `json:"reorderLevel" validate:"omitempty,min=0"`
	ReferenceType *string   `json:"referenceType"`
	ReferenceID   *string   `json:"referenceId"`
	Note          *string   `json:"note"`
}

type CreateReservationInput struct {
	WarehouseID   uuid.UUID                `json:"warehouseId" validate:"required"`
	ReferenceType string                   `json:"referenceType" validate:"required,max=100"`
	ReferenceID   string                   `json:"referenceId" validate:"required,max=150"`
	ExpiresAt     time.Time                `json:"expiresAt" validate:"required"`
	Items         []domain.ReservationItem `json:"items" validate:"required,min=1,dive"`
}

type Repository interface {
	CreateCategory(context.Context, domain.Category) (domain.Category, error)
	ListCategories(context.Context) ([]domain.Category, error)
	GetCategory(context.Context, uuid.UUID) (domain.Category, error)
	UpdateCategory(context.Context, uuid.UUID, UpdateCategoryInput) (domain.Category, error)
	DeleteCategory(context.Context, uuid.UUID) error

	CreateProduct(context.Context, domain.Product) (domain.Product, error)
	ListProducts(context.Context, ListFilter) (Page[domain.Product], error)
	GetProduct(context.Context, uuid.UUID) (domain.Product, error)
	UpdateProduct(context.Context, uuid.UUID, UpdateProductInput, uuid.UUID) (domain.Product, error)
	DeleteProduct(context.Context, uuid.UUID, uuid.UUID) error
	AssignCategory(context.Context, uuid.UUID, AssignCategoryInput) (domain.ProductCategory, error)
	ListProductCategories(context.Context, uuid.UUID) ([]domain.ProductCategory, error)
	RemoveCategory(context.Context, uuid.UUID, uuid.UUID) error
	CreateImage(context.Context, domain.ProductImage) (domain.ProductImage, error)
	ListImages(context.Context, uuid.UUID) ([]domain.ProductImage, error)
	UpdateImage(context.Context, uuid.UUID, uuid.UUID, UpdateImageInput) (domain.ProductImage, error)
	DeleteImage(context.Context, uuid.UUID, uuid.UUID) error

	CreateSKU(context.Context, domain.SKU) (domain.SKU, error)
	ListSKUs(context.Context, uuid.UUID) ([]domain.SKU, error)
	GetSKU(context.Context, uuid.UUID) (domain.SKU, error)
	UpdateSKU(context.Context, uuid.UUID, UpdateSKUInput) (domain.SKU, error)
	DeleteSKU(context.Context, uuid.UUID) error

	CreatePrice(context.Context, domain.Price) (domain.Price, error)
	ListPrices(context.Context, uuid.UUID) ([]domain.Price, error)
	UpdatePrice(context.Context, uuid.UUID, uuid.UUID, UpdatePriceInput) (domain.Price, error)
	DeletePrice(context.Context, uuid.UUID, uuid.UUID) error

	CreateWarehouse(context.Context, domain.Warehouse) (domain.Warehouse, error)
	ListWarehouses(context.Context) ([]domain.Warehouse, error)
	UpdateWarehouse(context.Context, uuid.UUID, UpdateWarehouseInput) (domain.Warehouse, error)
	DeleteWarehouse(context.Context, uuid.UUID) error

	ListInventory(context.Context, ListFilter) (Page[domain.Stock], error)
	GetStock(context.Context, uuid.UUID, uuid.UUID) (domain.Stock, error)
	AdjustStock(context.Context, AdjustmentInput, uuid.UUID) (domain.Stock, error)
	ListMovements(context.Context, ListFilter) (Page[domain.StockMovement], error)

	CreateReservation(context.Context, CreateReservationInput, uuid.UUID) (domain.Reservation, error)
	ListReservations(context.Context, ListFilter) (Page[domain.Reservation], error)
	GetReservation(context.Context, uuid.UUID) (domain.Reservation, error)
	ConfirmReservation(context.Context, uuid.UUID, uuid.UUID) (domain.Reservation, error)
	ReleaseReservation(context.Context, uuid.UUID, uuid.UUID) (domain.Reservation, error)
	ExpirePending(context.Context, time.Time, int) (int, error)
}
