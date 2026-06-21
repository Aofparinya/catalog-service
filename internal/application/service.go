package application

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/saaof/order-platform/catalog-service/internal/domain"
)

type FileValidator interface {
	ValidateImage(context.Context, uuid.UUID) error
}

type Service struct {
	repository Repository
	files      FileValidator
}

func NewService(repository Repository, validators ...FileValidator) *Service {
	var files FileValidator
	if len(validators) > 0 {
		files = validators[0]
	}
	return &Service{repository: repository, files: files}
}

func (service *Service) CreateCategory(ctx context.Context, input CreateCategoryInput) (domain.Category, error) {
	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	input.Name = strings.TrimSpace(input.Name)
	input.Description = trimPointer(input.Description)
	if input.Status == "" {
		input.Status = domain.StatusActive
	}
	category := domain.Category{
		ID:          uuid.New(),
		Code:        input.Code,
		Name:        input.Name,
		Description: input.Description,
		ParentID:    input.ParentID,
		Status:      input.Status,
	}
	if category.ParentID != nil && *category.ParentID == category.ID {
		return domain.Category{}, domain.ValidationError{Message: "category cannot be its own parent"}
	}
	return service.repository.CreateCategory(ctx, category)
}

func (service *Service) ListCategories(ctx context.Context) ([]domain.Category, error) {
	return service.repository.ListCategories(ctx)
}

func (service *Service) GetCategory(ctx context.Context, id uuid.UUID) (domain.Category, error) {
	return service.repository.GetCategory(ctx, id)
}

func (service *Service) UpdateCategory(ctx context.Context, id uuid.UUID, input UpdateCategoryInput) (domain.Category, error) {
	if input.ParentID != nil && *input.ParentID == id {
		return domain.Category{}, domain.ValidationError{Message: "category cannot be its own parent"}
	}
	if input.Code != nil {
		value := strings.ToUpper(strings.TrimSpace(*input.Code))
		input.Code = &value
	}
	if input.Name != nil {
		value := strings.TrimSpace(*input.Name)
		input.Name = &value
	}
	input.Description = trimPointer(input.Description)
	return service.repository.UpdateCategory(ctx, id, input)
}

func (service *Service) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	return service.repository.DeleteCategory(ctx, id)
}

func (service *Service) CreateProduct(ctx context.Context, input CreateProductInput, actorID uuid.UUID) (domain.Product, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Description = trimPointer(input.Description)
	if input.Status == "" {
		input.Status = domain.ProductStatusDraft
	}
	return service.repository.CreateProduct(ctx, domain.Product{
		ID:          uuid.New(),
		Name:        input.Name,
		Description: input.Description,
		Status:      input.Status,
		CreatedBy:   &actorID,
		UpdatedBy:   &actorID,
	})
}

func (service *Service) ListProducts(ctx context.Context, filter ListFilter) (Page[domain.Product], error) {
	normalizePage(&filter)
	filter.Query = strings.TrimSpace(filter.Query)
	return service.repository.ListProducts(ctx, filter)
}

func (service *Service) GetProduct(ctx context.Context, id uuid.UUID) (domain.Product, error) {
	return service.repository.GetProduct(ctx, id)
}

func (service *Service) UpdateProduct(ctx context.Context, id uuid.UUID, input UpdateProductInput, actorID uuid.UUID) (domain.Product, error) {
	if input.Name != nil {
		value := strings.TrimSpace(*input.Name)
		input.Name = &value
	}
	input.Description = trimPointer(input.Description)
	return service.repository.UpdateProduct(ctx, id, input, actorID)
}

func (service *Service) DeleteProduct(ctx context.Context, id uuid.UUID, actorID uuid.UUID) error {
	return service.repository.DeleteProduct(ctx, id, actorID)
}

func (service *Service) AssignCategory(ctx context.Context, productID uuid.UUID, input AssignCategoryInput) (domain.ProductCategory, error) {
	return service.repository.AssignCategory(ctx, productID, input)
}

func (service *Service) ListProductCategories(ctx context.Context, productID uuid.UUID) ([]domain.ProductCategory, error) {
	if _, err := service.repository.GetProduct(ctx, productID); err != nil {
		return nil, err
	}
	return service.repository.ListProductCategories(ctx, productID)
}

func (service *Service) RemoveCategory(ctx context.Context, productID, categoryID uuid.UUID) error {
	return service.repository.RemoveCategory(ctx, productID, categoryID)
}

func (service *Service) CreateImage(ctx context.Context, productID uuid.UUID, input CreateImageInput) (domain.ProductImage, error) {
	if _, err := service.repository.GetProduct(ctx, productID); err != nil {
		return domain.ProductImage{}, err
	}
	if service.files != nil {
		if err := service.files.ValidateImage(ctx, input.FileID); err != nil {
			return domain.ProductImage{}, err
		}
	}
	return service.repository.CreateImage(ctx, domain.ProductImage{
		ID:        uuid.New(),
		ProductID: productID,
		FileID:    input.FileID,
		AltText:   trimPointer(input.AltText),
		SortOrder: input.SortOrder,
		IsPrimary: input.IsPrimary,
	})
}

func (service *Service) ListImages(ctx context.Context, productID uuid.UUID) ([]domain.ProductImage, error) {
	if _, err := service.repository.GetProduct(ctx, productID); err != nil {
		return nil, err
	}
	return service.repository.ListImages(ctx, productID)
}

func (service *Service) UpdateImage(ctx context.Context, productID, imageID uuid.UUID, input UpdateImageInput) (domain.ProductImage, error) {
	input.AltText = trimPointer(input.AltText)
	return service.repository.UpdateImage(ctx, productID, imageID, input)
}

func (service *Service) DeleteImage(ctx context.Context, productID, imageID uuid.UUID) error {
	return service.repository.DeleteImage(ctx, productID, imageID)
}

func (service *Service) CreateSKU(ctx context.Context, productID uuid.UUID, input CreateSKUInput) (domain.SKU, error) {
	if _, err := service.repository.GetProduct(ctx, productID); err != nil {
		return domain.SKU{}, err
	}
	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	input.Name = strings.TrimSpace(input.Name)
	input.Barcode = trimPointer(input.Barcode)
	if input.Status == "" {
		input.Status = domain.StatusActive
	}
	attributes, err := normalizeAttributes(input.Attributes)
	if err != nil {
		return domain.SKU{}, err
	}
	return service.repository.CreateSKU(ctx, domain.SKU{
		ID:         uuid.New(),
		ProductID:  productID,
		Code:       input.Code,
		Barcode:    input.Barcode,
		Name:       input.Name,
		Attributes: attributes,
		Status:     input.Status,
	})
}

func (service *Service) ListSKUs(ctx context.Context, productID uuid.UUID) ([]domain.SKU, error) {
	if _, err := service.repository.GetProduct(ctx, productID); err != nil {
		return nil, err
	}
	return service.repository.ListSKUs(ctx, productID)
}

func (service *Service) GetSKU(ctx context.Context, id uuid.UUID) (domain.SKU, error) {
	return service.repository.GetSKU(ctx, id)
}

func (service *Service) UpdateSKU(ctx context.Context, id uuid.UUID, input UpdateSKUInput) (domain.SKU, error) {
	if input.Code != nil {
		value := strings.ToUpper(strings.TrimSpace(*input.Code))
		input.Code = &value
	}
	if input.Name != nil {
		value := strings.TrimSpace(*input.Name)
		input.Name = &value
	}
	input.Barcode = trimPointer(input.Barcode)
	if input.Attributes != nil {
		attributes, err := normalizeAttributes(*input.Attributes)
		if err != nil {
			return domain.SKU{}, err
		}
		input.Attributes = &attributes
	}
	return service.repository.UpdateSKU(ctx, id, input)
}

func (service *Service) DeleteSKU(ctx context.Context, id uuid.UUID) error {
	return service.repository.DeleteSKU(ctx, id)
}

func (service *Service) CreatePrice(ctx context.Context, skuID uuid.UUID, input CreatePriceInput) (domain.Price, error) {
	if _, err := service.repository.GetSKU(ctx, skuID); err != nil {
		return domain.Price{}, err
	}
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	if input.Amount.IsNegative() {
		return domain.Price{}, domain.ValidationError{Message: "amount cannot be negative"}
	}
	if input.ValidTo != nil && !input.ValidTo.After(input.ValidFrom) {
		return domain.Price{}, domain.ValidationError{Message: "validTo must be after validFrom"}
	}
	return service.repository.CreatePrice(ctx, domain.Price{
		ID:        uuid.New(),
		SKUID:     skuID,
		Amount:    input.Amount,
		Currency:  input.Currency,
		ValidFrom: input.ValidFrom,
		ValidTo:   input.ValidTo,
	})
}

func (service *Service) ListPrices(ctx context.Context, skuID uuid.UUID) ([]domain.Price, error) {
	return service.repository.ListPrices(ctx, skuID)
}

func (service *Service) UpdatePrice(ctx context.Context, skuID, priceID uuid.UUID, input UpdatePriceInput) (domain.Price, error) {
	if input.Currency != nil {
		value := strings.ToUpper(strings.TrimSpace(*input.Currency))
		input.Currency = &value
	}
	if input.Amount != nil && input.Amount.IsNegative() {
		return domain.Price{}, domain.ValidationError{Message: "amount cannot be negative"}
	}
	return service.repository.UpdatePrice(ctx, skuID, priceID, input)
}

func (service *Service) DeletePrice(ctx context.Context, skuID, priceID uuid.UUID) error {
	return service.repository.DeletePrice(ctx, skuID, priceID)
}

func (service *Service) CreateWarehouse(ctx context.Context, input CreateWarehouseInput) (domain.Warehouse, error) {
	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	input.Name = strings.TrimSpace(input.Name)
	if input.Status == "" {
		input.Status = domain.StatusActive
	}
	return service.repository.CreateWarehouse(ctx, domain.Warehouse{
		ID:     uuid.New(),
		Code:   input.Code,
		Name:   input.Name,
		Status: input.Status,
	})
}

func (service *Service) ListWarehouses(ctx context.Context) ([]domain.Warehouse, error) {
	return service.repository.ListWarehouses(ctx)
}

func (service *Service) UpdateWarehouse(ctx context.Context, id uuid.UUID, input UpdateWarehouseInput) (domain.Warehouse, error) {
	if input.Code != nil {
		value := strings.ToUpper(strings.TrimSpace(*input.Code))
		input.Code = &value
	}
	if input.Name != nil {
		value := strings.TrimSpace(*input.Name)
		input.Name = &value
	}
	return service.repository.UpdateWarehouse(ctx, id, input)
}

func (service *Service) DeleteWarehouse(ctx context.Context, id uuid.UUID) error {
	return service.repository.DeleteWarehouse(ctx, id)
}

func (service *Service) ListInventory(ctx context.Context, filter ListFilter) (Page[domain.Stock], error) {
	normalizePage(&filter)
	return service.repository.ListInventory(ctx, filter)
}

func (service *Service) GetStock(ctx context.Context, warehouseID, skuID uuid.UUID) (domain.Stock, error) {
	return service.repository.GetStock(ctx, warehouseID, skuID)
}

func (service *Service) AdjustStock(ctx context.Context, input AdjustmentInput, actorID uuid.UUID) (domain.Stock, error) {
	if input.Quantity == 0 && input.ReorderLevel == nil {
		return domain.Stock{}, domain.ValidationError{Message: "quantity or reorderLevel change is required"}
	}
	return service.repository.AdjustStock(ctx, input, actorID)
}

func (service *Service) ListMovements(ctx context.Context, filter ListFilter) (Page[domain.StockMovement], error) {
	normalizePage(&filter)
	return service.repository.ListMovements(ctx, filter)
}

func (service *Service) CreateReservation(ctx context.Context, input CreateReservationInput, actorID uuid.UUID) (domain.Reservation, error) {
	if !input.ExpiresAt.After(time.Now()) {
		return domain.Reservation{}, domain.ValidationError{Message: "expiresAt must be in the future"}
	}
	input.ReferenceType = strings.ToUpper(strings.TrimSpace(input.ReferenceType))
	input.ReferenceID = strings.TrimSpace(input.ReferenceID)
	if len(input.Items) == 0 {
		return domain.Reservation{}, domain.ValidationError{Message: "at least one reservation item is required"}
	}
	merged := make(map[uuid.UUID]int64)
	for _, item := range input.Items {
		if item.Quantity <= 0 {
			return domain.Reservation{}, domain.ValidationError{Message: "reservation quantity must be greater than zero"}
		}
		merged[item.SKUID] += item.Quantity
	}
	input.Items = make([]domain.ReservationItem, 0, len(merged))
	for skuID, quantity := range merged {
		input.Items = append(input.Items, domain.ReservationItem{SKUID: skuID, Quantity: quantity})
	}
	sort.Slice(input.Items, func(i, j int) bool { return input.Items[i].SKUID.String() < input.Items[j].SKUID.String() })
	return service.repository.CreateReservation(ctx, input, actorID)
}

func (service *Service) ListReservations(ctx context.Context, filter ListFilter) (Page[domain.Reservation], error) {
	normalizePage(&filter)
	filter.ReferenceType = strings.ToUpper(strings.TrimSpace(filter.ReferenceType))
	filter.ReferenceID = strings.TrimSpace(filter.ReferenceID)
	return service.repository.ListReservations(ctx, filter)
}

func (service *Service) GetReservation(ctx context.Context, id uuid.UUID) (domain.Reservation, error) {
	return service.repository.GetReservation(ctx, id)
}

func (service *Service) ConfirmReservation(ctx context.Context, id, actorID uuid.UUID) (domain.Reservation, error) {
	return service.repository.ConfirmReservation(ctx, id, actorID)
}

func (service *Service) ReleaseReservation(ctx context.Context, id, actorID uuid.UUID) (domain.Reservation, error) {
	return service.repository.ReleaseReservation(ctx, id, actorID)
}

func (service *Service) ExpirePending(ctx context.Context, now time.Time, limit int) (int, error) {
	return service.repository.ExpirePending(ctx, now, limit)
}

func NewPage[T any](data []T, page, pageSize int, total int64) Page[T] {
	totalPages := int64(0)
	if total > 0 {
		totalPages = int64(math.Ceil(float64(total) / float64(pageSize)))
	}
	return Page[T]{Data: data, Pagination: Pagination{
		Page: page, PageSize: pageSize, Total: total, TotalPages: totalPages,
	}}
}

func normalizePage(filter *ListFilter) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
}

func normalizeAttributes(value json.RawMessage) (json.RawMessage, error) {
	if len(value) == 0 {
		return json.RawMessage(`{}`), nil
	}
	var object map[string]any
	if err := json.Unmarshal(value, &object); err != nil {
		return nil, domain.ValidationError{Message: "attributes must be a JSON object"}
	}
	normalized, err := json.Marshal(object)
	if err != nil {
		return nil, err
	}
	return normalized, nil
}

func trimPointer(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
