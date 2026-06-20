package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/saaof/order-platform/catalog-service/internal/application"
	"github.com/saaof/order-platform/catalog-service/internal/domain"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) CreateCategory(ctx context.Context, value domain.Category) (domain.Category, error) {
	if value.ParentID != nil {
		if _, err := r.GetCategory(ctx, *value.ParentID); err != nil {
			return domain.Category{}, err
		}
	}
	model := categoryModel{ID: value.ID, Code: value.Code, Name: value.Name, Description: value.Description, ParentID: value.ParentID, Status: string(value.Status)}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return domain.Category{}, translate(err)
	}
	return categoryFromModel(model), nil
}
func (r *Repository) ListCategories(ctx context.Context) ([]domain.Category, error) {
	var models []categoryModel
	if err := r.db.WithContext(ctx).Order("name").Find(&models).Error; err != nil {
		return nil, translate(err)
	}
	result := make([]domain.Category, 0, len(models))
	for _, m := range models {
		result = append(result, categoryFromModel(m))
	}
	return result, nil
}
func (r *Repository) GetCategory(ctx context.Context, id uuid.UUID) (domain.Category, error) {
	var m categoryModel
	if err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		return domain.Category{}, translate(err)
	}
	return categoryFromModel(m), nil
}
func (r *Repository) UpdateCategory(ctx context.Context, id uuid.UUID, input application.UpdateCategoryInput) (domain.Category, error) {
	if input.ParentID != nil {
		if *input.ParentID == id {
			return domain.Category{}, domain.ValidationError{Message: "category cannot be its own parent"}
		}
		if _, err := r.GetCategory(ctx, *input.ParentID); err != nil {
			return domain.Category{}, err
		}
	}
	updates := map[string]any{"updated_at": time.Now().UTC()}
	set(updates, "code", input.Code)
	set(updates, "name", input.Name)
	set(updates, "description", input.Description)
	set(updates, "parent_id", input.ParentID)
	set(updates, "status", input.Status)
	result := r.db.WithContext(ctx).Model(&categoryModel{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return domain.Category{}, translate(result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.Category{}, domain.ErrNotFound
	}
	return r.GetCategory(ctx, id)
}
func (r *Repository) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	var children int64
	r.db.WithContext(ctx).Model(&categoryModel{}).Where("parent_id = ?", id).Count(&children)
	if children > 0 {
		return fmt.Errorf("%w: category has children", domain.ErrConflict)
	}
	var products int64
	r.db.WithContext(ctx).Model(&productCategoryModel{}).Where("category_id = ?", id).Count(&products)
	if products > 0 {
		return fmt.Errorf("%w: category is assigned to products", domain.ErrConflict)
	}
	result := r.db.WithContext(ctx).Model(&categoryModel{}).Where("id = ?", id).Update("deleted_at", time.Now().UTC())
	if result.Error != nil {
		return translate(result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) CreateProduct(ctx context.Context, value domain.Product) (domain.Product, error) {
	var result productModel
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var number string
		if err := tx.Raw(`SELECT 'PRD-' || to_char(CURRENT_DATE,'YYYYMMDD') || '-' || lpad(nextval('catalog.product_number_seq')::text,6,'0')`).Scan(&number).Error; err != nil {
			return err
		}
		result = productModel{ID: value.ID, ProductNo: number, Name: value.Name, Description: value.Description, Status: string(value.Status), CreatedBy: value.CreatedBy, UpdatedBy: value.UpdatedBy}
		return tx.Create(&result).Error
	})
	if err != nil {
		return domain.Product{}, translate(err)
	}
	return productFromModel(result), nil
}
func (r *Repository) ListProducts(ctx context.Context, filter application.ListFilter) (application.Page[domain.Product], error) {
	q := r.db.WithContext(ctx).Model(&productModel{})
	if filter.Query != "" {
		search := "%" + strings.ToLower(filter.Query) + "%"
		q = q.Where("lower(name) LIKE ? OR lower(product_no) LIKE ?", search, search)
	}
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.CategoryID != nil {
		q = q.Joins("JOIN catalog.product_categories pc ON pc.product_id = catalog.products.id").Where("pc.category_id = ?", *filter.CategoryID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return application.Page[domain.Product]{}, translate(err)
	}
	var models []productModel
	if err := q.Order("created_at DESC").Limit(filter.PageSize).Offset((filter.Page - 1) * filter.PageSize).Find(&models).Error; err != nil {
		return application.Page[domain.Product]{}, translate(err)
	}
	values := make([]domain.Product, 0, len(models))
	for _, m := range models {
		values = append(values, productFromModel(m))
	}
	return application.NewPage(values, filter.Page, filter.PageSize, total), nil
}
func (r *Repository) GetProduct(ctx context.Context, id uuid.UUID) (domain.Product, error) {
	var m productModel
	if err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		return domain.Product{}, translate(err)
	}
	return productFromModel(m), nil
}
func (r *Repository) UpdateProduct(ctx context.Context, id uuid.UUID, input application.UpdateProductInput, actor uuid.UUID) (domain.Product, error) {
	updates := map[string]any{"updated_at": time.Now().UTC(), "updated_by": actor}
	set(updates, "name", input.Name)
	set(updates, "description", input.Description)
	set(updates, "status", input.Status)
	result := r.db.WithContext(ctx).Model(&productModel{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return domain.Product{}, translate(result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.Product{}, domain.ErrNotFound
	}
	return r.GetProduct(ctx, id)
}
func (r *Repository) DeleteProduct(ctx context.Context, id, actor uuid.UUID) error {
	var skuCount int64
	r.db.WithContext(ctx).Model(&skuModel{}).Where("product_id = ?", id).Count(&skuCount)
	if skuCount > 0 {
		return fmt.Errorf("%w: product has SKUs", domain.ErrConflict)
	}
	result := r.db.WithContext(ctx).Model(&productModel{}).Where("id = ?", id).Updates(map[string]any{"deleted_at": time.Now().UTC(), "updated_at": time.Now().UTC(), "updated_by": actor})
	if result.Error != nil {
		return translate(result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}
func (r *Repository) AssignCategory(ctx context.Context, productID uuid.UUID, input application.AssignCategoryInput) (domain.ProductCategory, error) {
	if _, err := r.GetProduct(ctx, productID); err != nil {
		return domain.ProductCategory{}, err
	}
	if _, err := r.GetCategory(ctx, input.CategoryID); err != nil {
		return domain.ProductCategory{}, err
	}
	m := productCategoryModel{ProductID: productID, CategoryID: input.CategoryID, IsPrimary: input.IsPrimary}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if input.IsPrimary {
			if err := tx.Model(&productCategoryModel{}).Where("product_id = ?", productID).Update("is_primary", false).Error; err != nil {
				return err
			}
		}
		return tx.Save(&m).Error
	})
	if err != nil {
		return domain.ProductCategory{}, translate(err)
	}
	return domain.ProductCategory{ProductID: m.ProductID, CategoryID: m.CategoryID, IsPrimary: m.IsPrimary, CreatedAt: m.CreatedAt}, nil
}
func (r *Repository) RemoveCategory(ctx context.Context, productID, categoryID uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&productCategoryModel{}, "product_id = ? AND category_id = ?", productID, categoryID)
	if result.Error != nil {
		return translate(result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}
func (r *Repository) CreateImage(ctx context.Context, value domain.ProductImage) (domain.ProductImage, error) {
	m := productImageModel{ID: value.ID, ProductID: value.ProductID, FileID: value.FileID, AltText: value.AltText, SortOrder: value.SortOrder, IsPrimary: value.IsPrimary}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if m.IsPrimary {
			if err := tx.Model(&productImageModel{}).Where("product_id = ?", m.ProductID).Update("is_primary", false).Error; err != nil {
				return err
			}
		}
		return tx.Create(&m).Error
	})
	if err != nil {
		return domain.ProductImage{}, translate(err)
	}
	return imageFromModel(m), nil
}
func (r *Repository) UpdateImage(ctx context.Context, productID, imageID uuid.UUID, input application.UpdateImageInput) (domain.ProductImage, error) {
	var m productImageModel
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND product_id = ?", imageID, productID).First(&m).Error; err != nil {
			return err
		}
		if input.IsPrimary != nil && *input.IsPrimary {
			if err := tx.Model(&productImageModel{}).Where("product_id = ? AND id <> ?", productID, imageID).Update("is_primary", false).Error; err != nil {
				return err
			}
		}
		updates := map[string]any{"updated_at": time.Now().UTC()}
		set(updates, "alt_text", input.AltText)
		set(updates, "sort_order", input.SortOrder)
		set(updates, "is_primary", input.IsPrimary)
		if err := tx.Model(&m).Updates(updates).Error; err != nil {
			return err
		}
		return tx.First(&m, "id = ?", imageID).Error
	})
	if err != nil {
		return domain.ProductImage{}, translate(err)
	}
	return imageFromModel(m), nil
}
func (r *Repository) DeleteImage(ctx context.Context, productID, imageID uuid.UUID) error {
	return r.softDelete(ctx, &productImageModel{}, map[string]any{"id": imageID, "product_id": productID})
}

func (r *Repository) CreateSKU(ctx context.Context, value domain.SKU) (domain.SKU, error) {
	m := skuModel{ID: value.ID, ProductID: value.ProductID, Code: value.Code, Barcode: value.Barcode, Name: value.Name, Attributes: value.Attributes, Status: string(value.Status)}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return domain.SKU{}, translate(err)
	}
	return skuFromModel(m), nil
}
func (r *Repository) ListSKUs(ctx context.Context, productID uuid.UUID) ([]domain.SKU, error) {
	var models []skuModel
	if err := r.db.WithContext(ctx).Where("product_id = ?", productID).Order("created_at").Find(&models).Error; err != nil {
		return nil, translate(err)
	}
	values := make([]domain.SKU, 0, len(models))
	for _, m := range models {
		values = append(values, skuFromModel(m))
	}
	return values, nil
}
func (r *Repository) GetSKU(ctx context.Context, id uuid.UUID) (domain.SKU, error) {
	var m skuModel
	if err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		return domain.SKU{}, translate(err)
	}
	return skuFromModel(m), nil
}
func (r *Repository) UpdateSKU(ctx context.Context, id uuid.UUID, input application.UpdateSKUInput) (domain.SKU, error) {
	updates := map[string]any{"updated_at": time.Now().UTC()}
	set(updates, "code", input.Code)
	set(updates, "barcode", input.Barcode)
	set(updates, "name", input.Name)
	set(updates, "attributes", input.Attributes)
	set(updates, "status", input.Status)
	result := r.db.WithContext(ctx).Model(&skuModel{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return domain.SKU{}, translate(result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.SKU{}, domain.ErrNotFound
	}
	return r.GetSKU(ctx, id)
}
func (r *Repository) DeleteSKU(ctx context.Context, id uuid.UUID) error {
	var stock int64
	r.db.WithContext(ctx).Model(&stockModel{}).Where("sku_id = ? AND (on_hand > 0 OR reserved > 0)", id).Count(&stock)
	if stock > 0 {
		return fmt.Errorf("%w: SKU has stock", domain.ErrConflict)
	}
	return r.softDelete(ctx, &skuModel{}, map[string]any{"id": id})
}

func (r *Repository) CreatePrice(ctx context.Context, value domain.Price) (domain.Price, error) {
	if err := r.checkPriceOverlap(ctx, r.db, value.SKUID, value.Currency, value.ValidFrom, value.ValidTo, uuid.Nil); err != nil {
		return domain.Price{}, err
	}
	m := priceModel{ID: value.ID, SKUID: value.SKUID, Amount: value.Amount, Currency: value.Currency, ValidFrom: value.ValidFrom, ValidTo: value.ValidTo}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return domain.Price{}, translate(err)
	}
	return priceFromModel(m), nil
}
func (r *Repository) ListPrices(ctx context.Context, skuID uuid.UUID) ([]domain.Price, error) {
	var models []priceModel
	if err := r.db.WithContext(ctx).Where("sku_id = ?", skuID).Order("valid_from DESC").Find(&models).Error; err != nil {
		return nil, translate(err)
	}
	values := make([]domain.Price, 0, len(models))
	for _, m := range models {
		values = append(values, priceFromModel(m))
	}
	return values, nil
}
func (r *Repository) UpdatePrice(ctx context.Context, skuID, priceID uuid.UUID, input application.UpdatePriceInput) (domain.Price, error) {
	var m priceModel
	if err := r.db.WithContext(ctx).Where("id = ? AND sku_id = ?", priceID, skuID).First(&m).Error; err != nil {
		return domain.Price{}, translate(err)
	}
	currency := m.Currency
	if input.Currency != nil {
		currency = *input.Currency
	}
	from := m.ValidFrom
	if input.ValidFrom != nil {
		from = *input.ValidFrom
	}
	to := m.ValidTo
	if input.ValidTo != nil {
		to = input.ValidTo
	}
	if to != nil && !to.After(from) {
		return domain.Price{}, domain.ValidationError{Message: "validTo must be after validFrom"}
	}
	if err := r.checkPriceOverlap(ctx, r.db, skuID, currency, from, to, priceID); err != nil {
		return domain.Price{}, err
	}
	updates := map[string]any{"updated_at": time.Now().UTC()}
	set(updates, "amount", input.Amount)
	set(updates, "currency", input.Currency)
	set(updates, "valid_from", input.ValidFrom)
	set(updates, "valid_to", input.ValidTo)
	if err := r.db.WithContext(ctx).Model(&m).Updates(updates).Error; err != nil {
		return domain.Price{}, translate(err)
	}
	if err := r.db.WithContext(ctx).First(&m, "id = ?", priceID).Error; err != nil {
		return domain.Price{}, translate(err)
	}
	return priceFromModel(m), nil
}
func (r *Repository) DeletePrice(ctx context.Context, skuID, priceID uuid.UUID) error {
	return r.softDelete(ctx, &priceModel{}, map[string]any{"id": priceID, "sku_id": skuID})
}
func (r *Repository) checkPriceOverlap(ctx context.Context, db *gorm.DB, skuID uuid.UUID, currency string, from time.Time, to *time.Time, exclude uuid.UUID) error {
	query := db.WithContext(ctx).Model(&priceModel{}).Where("sku_id = ? AND currency = ?", skuID, currency).Where("valid_from < COALESCE(?, 'infinity'::timestamptz) AND COALESCE(valid_to, 'infinity'::timestamptz) > ?", to, from)
	if exclude != uuid.Nil {
		query = query.Where("id <> ?", exclude)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return translate(err)
	}
	if count > 0 {
		return fmt.Errorf("%w: price period overlaps an existing price", domain.ErrConflict)
	}
	return nil
}

func (r *Repository) CreateWarehouse(ctx context.Context, value domain.Warehouse) (domain.Warehouse, error) {
	m := warehouseModel{ID: value.ID, Code: value.Code, Name: value.Name, Status: string(value.Status)}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return domain.Warehouse{}, translate(err)
	}
	return warehouseFromModel(m), nil
}
func (r *Repository) ListWarehouses(ctx context.Context) ([]domain.Warehouse, error) {
	var models []warehouseModel
	if err := r.db.WithContext(ctx).Order("name").Find(&models).Error; err != nil {
		return nil, translate(err)
	}
	values := make([]domain.Warehouse, 0, len(models))
	for _, m := range models {
		values = append(values, warehouseFromModel(m))
	}
	return values, nil
}
func (r *Repository) UpdateWarehouse(ctx context.Context, id uuid.UUID, input application.UpdateWarehouseInput) (domain.Warehouse, error) {
	updates := map[string]any{"updated_at": time.Now().UTC()}
	set(updates, "code", input.Code)
	set(updates, "name", input.Name)
	set(updates, "status", input.Status)
	result := r.db.WithContext(ctx).Model(&warehouseModel{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return domain.Warehouse{}, translate(result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.Warehouse{}, domain.ErrNotFound
	}
	var m warehouseModel
	if err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		return domain.Warehouse{}, translate(err)
	}
	return warehouseFromModel(m), nil
}
func (r *Repository) DeleteWarehouse(ctx context.Context, id uuid.UUID) error {
	var stock int64
	r.db.WithContext(ctx).Model(&stockModel{}).Where("warehouse_id = ? AND (on_hand > 0 OR reserved > 0)", id).Count(&stock)
	if stock > 0 {
		return fmt.Errorf("%w: warehouse has stock", domain.ErrConflict)
	}
	return r.softDelete(ctx, &warehouseModel{}, map[string]any{"id": id})
}

func (r *Repository) softDelete(ctx context.Context, model any, where map[string]any) error {
	result := r.db.WithContext(ctx).Model(model).Where(where).Update("deleted_at", time.Now().UTC())
	if result.Error != nil {
		return translate(result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}
func translate(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrConflict) || errors.Is(err, domain.ErrInvalid) || errors.Is(err, domain.ErrInsufficientStock) {
		return err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.ErrNotFound
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "23505":
			return fmt.Errorf("%w: duplicate value", domain.ErrConflict)
		case "23514", "23502", "23503":
			return domain.ValidationError{Message: pg.Message}
		}
	}
	return err
}
func set[T any](updates map[string]any, key string, value *T) {
	if value != nil {
		updates[key] = *value
	}
}

func categoryFromModel(m categoryModel) domain.Category {
	return domain.Category{ID: m.ID, Code: m.Code, Name: m.Name, Description: m.Description, ParentID: m.ParentID, Status: domain.Status(m.Status), CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}
}
func productFromModel(m productModel) domain.Product {
	return domain.Product{ID: m.ID, ProductNo: m.ProductNo, Name: m.Name, Description: m.Description, Status: domain.ProductStatus(m.Status), CreatedBy: m.CreatedBy, UpdatedBy: m.UpdatedBy, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}
}
func imageFromModel(m productImageModel) domain.ProductImage {
	return domain.ProductImage{ID: m.ID, ProductID: m.ProductID, FileID: m.FileID, AltText: m.AltText, SortOrder: m.SortOrder, IsPrimary: m.IsPrimary, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}
}
func skuFromModel(m skuModel) domain.SKU {
	return domain.SKU{ID: m.ID, ProductID: m.ProductID, Code: m.Code, Barcode: m.Barcode, Name: m.Name, Attributes: m.Attributes, Status: domain.Status(m.Status), CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}
}
func priceFromModel(m priceModel) domain.Price {
	return domain.Price{ID: m.ID, SKUID: m.SKUID, Amount: m.Amount, Currency: m.Currency, ValidFrom: m.ValidFrom, ValidTo: m.ValidTo, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}
}
func warehouseFromModel(m warehouseModel) domain.Warehouse {
	return domain.Warehouse{ID: m.ID, Code: m.Code, Name: m.Name, Status: domain.Status(m.Status), CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}
}
