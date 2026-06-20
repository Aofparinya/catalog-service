package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saaof/order-platform/catalog-service/internal/application"
	"github.com/saaof/order-platform/catalog-service/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *Repository) ListInventory(ctx context.Context, filter application.ListFilter) (application.Page[domain.Stock], error) {
	query := r.db.WithContext(ctx).Model(&stockModel{})
	if filter.WarehouseID != nil {
		query = query.Where("warehouse_id = ?", *filter.WarehouseID)
	}
	if filter.SKUID != nil {
		query = query.Where("sku_id = ?", *filter.SKUID)
	}
	if filter.LowStock {
		query = query.Where("(on_hand - reserved) <= reorder_level")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return application.Page[domain.Stock]{}, translate(err)
	}
	var models []stockModel
	if err := query.Order("updated_at DESC").
		Limit(filter.PageSize).
		Offset((filter.Page - 1) * filter.PageSize).
		Find(&models).Error; err != nil {
		return application.Page[domain.Stock]{}, translate(err)
	}
	values := make([]domain.Stock, 0, len(models))
	for _, model := range models {
		values = append(values, stockFromModel(model))
	}
	return application.NewPage(values, filter.Page, filter.PageSize, total), nil
}

func (r *Repository) GetStock(ctx context.Context, warehouseID, skuID uuid.UUID) (domain.Stock, error) {
	var model stockModel
	if err := r.db.WithContext(ctx).
		Where("warehouse_id = ? AND sku_id = ?", warehouseID, skuID).
		First(&model).Error; err != nil {
		return domain.Stock{}, translate(err)
	}
	return stockFromModel(model), nil
}

func (r *Repository) AdjustStock(ctx context.Context, input application.AdjustmentInput, actorID uuid.UUID) (domain.Stock, error) {
	var result stockModel
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureWarehouseAndSKU(tx, input.WarehouseID, input.SKUID); err != nil {
			return err
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&stockModel{
			WarehouseID: input.WarehouseID,
			SKUID:       input.SKUID,
		}).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("warehouse_id = ? AND sku_id = ?", input.WarehouseID, input.SKUID).
			First(&result).Error; err != nil {
			return err
		}
		newOnHand := result.OnHand + input.Quantity
		reorderLevel := result.ReorderLevel
		if input.ReorderLevel != nil {
			reorderLevel = *input.ReorderLevel
		}
		if newOnHand < 0 || newOnHand < result.Reserved {
			return domain.ErrInsufficientStock
		}
		if err := tx.Model(&result).Updates(map[string]any{
			"on_hand":       newOnHand,
			"reorder_level": reorderLevel,
			"updated_at":    time.Now().UTC(),
		}).Error; err != nil {
			return err
		}
		result.OnHand = newOnHand
		result.ReorderLevel = reorderLevel
		return createMovement(tx, movementModel{
			ID:            uuid.New(),
			WarehouseID:   input.WarehouseID,
			SKUID:         input.SKUID,
			MovementType:  "ADJUSTMENT",
			OnHandChange:  input.Quantity,
			OnHandAfter:   result.OnHand,
			ReservedAfter: result.Reserved,
			ReferenceType: input.ReferenceType,
			ReferenceID:   input.ReferenceID,
			Note:          input.Note,
			CreatedBy:     &actorID,
		})
	})
	if err != nil {
		return domain.Stock{}, translate(err)
	}
	return stockFromModel(result), nil
}

func (r *Repository) ListMovements(ctx context.Context, filter application.ListFilter) (application.Page[domain.StockMovement], error) {
	query := r.db.WithContext(ctx).Model(&movementModel{})
	if filter.WarehouseID != nil {
		query = query.Where("warehouse_id = ?", *filter.WarehouseID)
	}
	if filter.SKUID != nil {
		query = query.Where("sku_id = ?", *filter.SKUID)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return application.Page[domain.StockMovement]{}, translate(err)
	}
	var models []movementModel
	if err := query.Order("created_at DESC").
		Limit(filter.PageSize).
		Offset((filter.Page - 1) * filter.PageSize).
		Find(&models).Error; err != nil {
		return application.Page[domain.StockMovement]{}, translate(err)
	}
	values := make([]domain.StockMovement, 0, len(models))
	for _, model := range models {
		values = append(values, movementFromModel(model))
	}
	return application.NewPage(values, filter.Page, filter.PageSize, total), nil
}

func (r *Repository) CreateReservation(
	ctx context.Context,
	input application.CreateReservationInput,
	actorID uuid.UUID,
) (domain.Reservation, error) {
	requestHash, err := reservationHash(input)
	if err != nil {
		return domain.Reservation{}, err
	}
	var result reservationModel
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing reservationModel
		findError := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("reference_type = ? AND reference_id = ?", input.ReferenceType, input.ReferenceID).
			First(&existing).Error
		if findError == nil {
			if existing.RequestHash != requestHash {
				return fmt.Errorf("%w: reference already used with a different payload", domain.ErrConflict)
			}
			result = existing
			return nil
		}
		if !errors.Is(findError, gorm.ErrRecordNotFound) {
			return findError
		}
		if err := ensureWarehouse(tx, input.WarehouseID); err != nil {
			return err
		}
		result = reservationModel{
			ID:            uuid.New(),
			WarehouseID:   input.WarehouseID,
			ReferenceType: input.ReferenceType,
			ReferenceID:   input.ReferenceID,
			RequestHash:   requestHash,
			Status:        string(domain.ReservationPending),
			ExpiresAt:     input.ExpiresAt,
			CreatedBy:     &actorID,
		}
		if err := tx.Create(&result).Error; err != nil {
			return err
		}
		for _, item := range input.Items {
			stock, err := lockStock(tx, input.WarehouseID, item.SKUID)
			if err != nil {
				return err
			}
			if stock.OnHand-stock.Reserved < item.Quantity {
				return fmt.Errorf("%w: sku %s", domain.ErrInsufficientStock, item.SKUID)
			}
			stock.Reserved += item.Quantity
			if err := tx.Model(&stock).Updates(map[string]any{
				"reserved":   stock.Reserved,
				"updated_at": time.Now().UTC(),
			}).Error; err != nil {
				return err
			}
			if err := tx.Create(&reservationItemModel{
				ReservationID: result.ID,
				SKUID:         item.SKUID,
				Quantity:      item.Quantity,
			}).Error; err != nil {
				return err
			}
			if err := createMovement(tx, movementModel{
				ID:             uuid.New(),
				WarehouseID:    input.WarehouseID,
				SKUID:          item.SKUID,
				MovementType:   "RESERVE",
				ReservedChange: item.Quantity,
				OnHandAfter:    stock.OnHand,
				ReservedAfter:  stock.Reserved,
				ReferenceType:  &input.ReferenceType,
				ReferenceID:    &input.ReferenceID,
				CreatedBy:      &actorID,
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		var existing reservationModel
		findError := r.db.WithContext(ctx).
			Where("reference_type = ? AND reference_id = ?", input.ReferenceType, input.ReferenceID).
			First(&existing).Error
		if findError == nil {
			if existing.RequestHash != requestHash {
				return domain.Reservation{}, fmt.Errorf(
					"%w: reference already used with a different payload",
					domain.ErrConflict,
				)
			}
			return r.GetReservation(ctx, existing.ID)
		}
		return domain.Reservation{}, translate(err)
	}
	return r.GetReservation(ctx, result.ID)
}

func (r *Repository) ListReservations(ctx context.Context, filter application.ListFilter) (application.Page[domain.Reservation], error) {
	query := r.db.WithContext(ctx).Model(&reservationModel{})
	if filter.WarehouseID != nil {
		query = query.Where("warehouse_id = ?", *filter.WarehouseID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.ReferenceType != "" {
		query = query.Where("reference_type = ?", filter.ReferenceType)
	}
	if filter.ReferenceID != "" {
		query = query.Where("reference_id ILIKE ?", "%"+filter.ReferenceID+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return application.Page[domain.Reservation]{}, translate(err)
	}
	var models []reservationModel
	if err := query.Order("created_at DESC").
		Limit(filter.PageSize).
		Offset((filter.Page - 1) * filter.PageSize).
		Find(&models).Error; err != nil {
		return application.Page[domain.Reservation]{}, translate(err)
	}
	values := make([]domain.Reservation, 0, len(models))
	for _, model := range models {
		reservation, err := r.reservationFromModel(ctx, model)
		if err != nil {
			return application.Page[domain.Reservation]{}, err
		}
		values = append(values, reservation)
	}
	return application.NewPage(values, filter.Page, filter.PageSize, total), nil
}

func (r *Repository) GetReservation(ctx context.Context, id uuid.UUID) (domain.Reservation, error) {
	var model reservationModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		return domain.Reservation{}, translate(err)
	}
	if model.Status == string(domain.ReservationPending) && !model.ExpiresAt.After(time.Now()) {
		if _, err := r.expireReservation(ctx, id); err != nil {
			return domain.Reservation{}, err
		}
		if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
			return domain.Reservation{}, translate(err)
		}
	}
	return r.reservationFromModel(ctx, model)
}

func (r *Repository) ConfirmReservation(ctx context.Context, id, actorID uuid.UUID) (domain.Reservation, error) {
	var result reservationModel
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&result, "id = ?", id).Error; err != nil {
			return err
		}
		if result.Status == string(domain.ReservationConfirmed) {
			return nil
		}
		if result.Status != string(domain.ReservationPending) {
			return fmt.Errorf("%w: reservation cannot be confirmed from %s", domain.ErrConflict, result.Status)
		}
		if !result.ExpiresAt.After(time.Now()) {
			return r.expireReservationTx(tx, &result, actorID)
		}
		items, err := reservationItems(tx, result.ID)
		if err != nil {
			return err
		}
		for _, item := range items {
			stock, err := lockStock(tx, result.WarehouseID, item.SKUID)
			if err != nil {
				return err
			}
			if stock.Reserved < item.Quantity || stock.OnHand < item.Quantity {
				return domain.ErrInsufficientStock
			}
			stock.OnHand -= item.Quantity
			stock.Reserved -= item.Quantity
			if err := tx.Model(&stock).Updates(map[string]any{
				"on_hand":    stock.OnHand,
				"reserved":   stock.Reserved,
				"updated_at": time.Now().UTC(),
			}).Error; err != nil {
				return err
			}
			if err := createMovement(tx, movementModel{
				ID:             uuid.New(),
				WarehouseID:    result.WarehouseID,
				SKUID:          item.SKUID,
				MovementType:   "CONFIRM",
				OnHandChange:   -item.Quantity,
				ReservedChange: -item.Quantity,
				OnHandAfter:    stock.OnHand,
				ReservedAfter:  stock.Reserved,
				ReferenceType:  &result.ReferenceType,
				ReferenceID:    &result.ReferenceID,
				CreatedBy:      &actorID,
			}); err != nil {
				return err
			}
		}
		now := time.Now().UTC()
		result.Status = string(domain.ReservationConfirmed)
		result.ConfirmedAt = &now
		return tx.Model(&result).Updates(map[string]any{
			"status":       result.Status,
			"confirmed_at": now,
			"updated_at":   now,
		}).Error
	})
	if err != nil {
		return domain.Reservation{}, translate(err)
	}
	return r.reservationFromModel(ctx, result)
}

func (r *Repository) ReleaseReservation(ctx context.Context, id, actorID uuid.UUID) (domain.Reservation, error) {
	var result reservationModel
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&result, "id = ?", id).Error; err != nil {
			return err
		}
		if result.Status == string(domain.ReservationReleased) || result.Status == string(domain.ReservationExpired) {
			return nil
		}
		if result.Status != string(domain.ReservationPending) {
			return fmt.Errorf("%w: confirmed reservation cannot be released", domain.ErrConflict)
		}
		return r.releaseReservationTx(tx, &result, domain.ReservationReleased, "RELEASE", actorID)
	})
	if err != nil {
		return domain.Reservation{}, translate(err)
	}
	return r.reservationFromModel(ctx, result)
}

func (r *Repository) ExpirePending(ctx context.Context, now time.Time, limit int) (int, error) {
	var ids []uuid.UUID
	if err := r.db.WithContext(ctx).Model(&reservationModel{}).
		Where("status = ? AND expires_at <= ?", domain.ReservationPending, now).
		Order("expires_at").
		Limit(limit).
		Pluck("id", &ids).Error; err != nil {
		return 0, translate(err)
	}
	expired := 0
	for _, id := range ids {
		changed, err := r.expireReservation(ctx, id)
		if err != nil {
			return expired, err
		}
		if changed {
			expired++
		}
	}
	return expired, nil
}

func (r *Repository) expireReservation(ctx context.Context, id uuid.UUID) (bool, error) {
	changed := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var model reservationModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&model, "id = ?", id).Error; err != nil {
			return err
		}
		if model.Status != string(domain.ReservationPending) || model.ExpiresAt.After(time.Now()) {
			return nil
		}
		changed = true
		return r.expireReservationTx(tx, &model, uuid.Nil)
	})
	return changed, translate(err)
}

func (r *Repository) expireReservationTx(tx *gorm.DB, model *reservationModel, actorID uuid.UUID) error {
	return r.releaseReservationTx(tx, model, domain.ReservationExpired, "EXPIRE", actorID)
}

func (r *Repository) releaseReservationTx(
	tx *gorm.DB,
	model *reservationModel,
	status domain.ReservationStatus,
	movementType string,
	actorID uuid.UUID,
) error {
	items, err := reservationItems(tx, model.ID)
	if err != nil {
		return err
	}
	for _, item := range items {
		stock, err := lockStock(tx, model.WarehouseID, item.SKUID)
		if err != nil {
			return err
		}
		if stock.Reserved < item.Quantity {
			return domain.ErrInsufficientStock
		}
		stock.Reserved -= item.Quantity
		if err := tx.Model(&stock).Updates(map[string]any{
			"reserved":   stock.Reserved,
			"updated_at": time.Now().UTC(),
		}).Error; err != nil {
			return err
		}
		var createdBy *uuid.UUID
		if actorID != uuid.Nil {
			createdBy = &actorID
		}
		if err := createMovement(tx, movementModel{
			ID:             uuid.New(),
			WarehouseID:    model.WarehouseID,
			SKUID:          item.SKUID,
			MovementType:   movementType,
			ReservedChange: -item.Quantity,
			OnHandAfter:    stock.OnHand,
			ReservedAfter:  stock.Reserved,
			ReferenceType:  &model.ReferenceType,
			ReferenceID:    &model.ReferenceID,
			CreatedBy:      createdBy,
		}); err != nil {
			return err
		}
	}
	now := time.Now().UTC()
	model.Status = string(status)
	model.ReleasedAt = &now
	return tx.Model(model).Updates(map[string]any{
		"status":      model.Status,
		"released_at": now,
		"updated_at":  now,
	}).Error
}

func (r *Repository) reservationFromModel(ctx context.Context, model reservationModel) (domain.Reservation, error) {
	items, err := reservationItems(r.db.WithContext(ctx), model.ID)
	if err != nil {
		return domain.Reservation{}, translate(err)
	}
	domainItems := make([]domain.ReservationItem, 0, len(items))
	for _, item := range items {
		domainItems = append(domainItems, domain.ReservationItem{SKUID: item.SKUID, Quantity: item.Quantity})
	}
	return domain.Reservation{
		ID:            model.ID,
		WarehouseID:   model.WarehouseID,
		ReferenceType: model.ReferenceType,
		ReferenceID:   model.ReferenceID,
		Status:        domain.ReservationStatus(model.Status),
		ExpiresAt:     model.ExpiresAt,
		ConfirmedAt:   model.ConfirmedAt,
		ReleasedAt:    model.ReleasedAt,
		Items:         domainItems,
		CreatedAt:     model.CreatedAt,
		UpdatedAt:     model.UpdatedAt,
	}, nil
}

func ensureWarehouseAndSKU(tx *gorm.DB, warehouseID, skuID uuid.UUID) error {
	if err := ensureWarehouse(tx, warehouseID); err != nil {
		return err
	}
	var count int64
	if err := tx.Model(&skuModel{}).Where("id = ? AND status = ?", skuID, domain.StatusActive).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func ensureWarehouse(tx *gorm.DB, warehouseID uuid.UUID) error {
	var count int64
	if err := tx.Model(&warehouseModel{}).Where("id = ? AND status = ?", warehouseID, domain.StatusActive).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func lockStock(tx *gorm.DB, warehouseID, skuID uuid.UUID) (stockModel, error) {
	if err := ensureWarehouseAndSKU(tx, warehouseID, skuID); err != nil {
		return stockModel{}, err
	}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&stockModel{
		WarehouseID: warehouseID,
		SKUID:       skuID,
	}).Error; err != nil {
		return stockModel{}, err
	}
	var stock stockModel
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("warehouse_id = ? AND sku_id = ?", warehouseID, skuID).
		First(&stock).Error; err != nil {
		return stockModel{}, err
	}
	return stock, nil
}

func reservationItems(db *gorm.DB, reservationID uuid.UUID) ([]reservationItemModel, error) {
	var items []reservationItemModel
	if err := db.Where("reservation_id = ?", reservationID).Order("sku_id").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func reservationHash(input application.CreateReservationInput) (string, error) {
	value, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("hash reservation request: %w", err)
	}
	hash := sha256.Sum256(value)
	return hex.EncodeToString(hash[:]), nil
}

func createMovement(tx *gorm.DB, movement movementModel) error {
	return tx.Create(&movement).Error
}

func stockFromModel(model stockModel) domain.Stock {
	return domain.Stock{
		WarehouseID:  model.WarehouseID,
		SKUID:        model.SKUID,
		OnHand:       model.OnHand,
		Reserved:     model.Reserved,
		Available:    model.OnHand - model.Reserved,
		ReorderLevel: model.ReorderLevel,
		UpdatedAt:    model.UpdatedAt,
	}
}

func movementFromModel(model movementModel) domain.StockMovement {
	return domain.StockMovement{
		ID:             model.ID,
		WarehouseID:    model.WarehouseID,
		SKUID:          model.SKUID,
		MovementType:   model.MovementType,
		OnHandChange:   model.OnHandChange,
		ReservedChange: model.ReservedChange,
		OnHandAfter:    model.OnHandAfter,
		ReservedAfter:  model.ReservedAfter,
		ReferenceType:  model.ReferenceType,
		ReferenceID:    model.ReferenceID,
		Note:           model.Note,
		CreatedBy:      model.CreatedBy,
		CreatedAt:      model.CreatedAt,
	}
}
