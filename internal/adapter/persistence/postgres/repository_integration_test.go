package postgres_test

import (
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	postgresrepository "github.com/saaof/order-platform/catalog-service/internal/adapter/persistence/postgres"
	"github.com/saaof/order-platform/catalog-service/internal/application"
	"github.com/saaof/order-platform/catalog-service/internal/domain"
	"github.com/saaof/order-platform/catalog-service/internal/infrastructure/database"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func TestReservationConcurrencyAndIdempotency(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	db, err := database.Open(databaseURL, true)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}

	repository := postgresrepository.NewRepository(db)
	service := application.NewService(repository)
	actorID := uuid.New()
	suffix := time.Now().UTC().Format("20060102150405.000000000")

	product, err := service.CreateProduct(t.Context(), application.CreateProductInput{
		Name:   "Integration Product " + suffix,
		Status: domain.ProductStatusActive,
	}, actorID)
	if err != nil {
		t.Fatal(err)
	}
	sku, err := service.CreateSKU(t.Context(), product.ID, application.CreateSKUInput{
		Code:   "IT-SKU-" + suffix,
		Name:   "Integration SKU",
		Status: domain.StatusActive,
	})
	if err != nil {
		t.Fatal(err)
	}
	warehouse, err := service.CreateWarehouse(t.Context(), application.CreateWarehouseInput{
		Code:   "IT-WH-" + suffix,
		Name:   "Integration Warehouse",
		Status: domain.StatusActive,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupFixture(t, db, warehouse.ID, product.ID)
	})
	if _, err := service.AdjustStock(t.Context(), application.AdjustmentInput{
		WarehouseID: warehouse.ID,
		SKUID:       sku.ID,
		Quantity:    5,
	}, actorID); err != nil {
		t.Fatal(err)
	}

	t.Run("price periods cannot overlap", func(t *testing.T) {
		validFrom := time.Now().UTC().Truncate(time.Second)
		validTo := validFrom.Add(time.Hour)
		if _, err := service.CreatePrice(t.Context(), sku.ID, application.CreatePriceInput{
			Amount:    decimal.NewFromInt(100),
			Currency:  "THB",
			ValidFrom: validFrom,
			ValidTo:   &validTo,
		}); err != nil {
			t.Fatal(err)
		}
		overlapTo := validTo.Add(time.Hour)
		_, err := service.CreatePrice(t.Context(), sku.ID, application.CreatePriceInput{
			Amount:    decimal.NewFromInt(120),
			Currency:  "THB",
			ValidFrom: validFrom.Add(30 * time.Minute),
			ValidTo:   &overlapTo,
		})
		if !errors.Is(err, domain.ErrConflict) {
			t.Fatalf("expected price overlap conflict, got %v", err)
		}
	})

	t.Run("row locking prevents overselling", func(t *testing.T) {
		start := make(chan struct{})
		results := make(chan reservationResult, 2)
		var waitGroup sync.WaitGroup
		for index := 0; index < 2; index++ {
			waitGroup.Add(1)
			go func(index int) {
				defer waitGroup.Done()
				<-start
				reservation, createErr := service.CreateReservation(
					t.Context(),
					application.CreateReservationInput{
						WarehouseID:   warehouse.ID,
						ReferenceType: "TEST",
						ReferenceID:   suffix + "-concurrent-" + string(rune('A'+index)),
						ExpiresAt:     time.Now().UTC().Add(time.Minute),
						Items: []domain.ReservationItem{{
							SKUID:    sku.ID,
							Quantity: 4,
						}},
					},
					actorID,
				)
				results <- reservationResult{reservation: reservation, err: createErr}
			}(index)
		}
		close(start)
		waitGroup.Wait()
		close(results)

		successes := make([]domain.Reservation, 0, 1)
		insufficient := 0
		for result := range results {
			switch {
			case result.err == nil:
				successes = append(successes, result.reservation)
			case errors.Is(result.err, domain.ErrInsufficientStock):
				insufficient++
			default:
				t.Fatalf("unexpected reservation error: %v", result.err)
			}
		}
		if len(successes) != 1 || insufficient != 1 {
			t.Fatalf("successes=%d insufficient=%d", len(successes), insufficient)
		}
		stock, err := service.GetStock(t.Context(), warehouse.ID, sku.ID)
		if err != nil {
			t.Fatal(err)
		}
		if stock.Available != 1 || stock.Reserved != 4 {
			t.Fatalf("unexpected stock after concurrent reserve: %+v", stock)
		}
		if _, err := service.ReleaseReservation(t.Context(), successes[0].ID, actorID); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("concurrent duplicate request returns same reservation", func(t *testing.T) {
		input := application.CreateReservationInput{
			WarehouseID:   warehouse.ID,
			ReferenceType: "TEST",
			ReferenceID:   suffix + "-idempotent",
			ExpiresAt:     time.Now().UTC().Add(time.Minute),
			Items: []domain.ReservationItem{{
				SKUID:    sku.ID,
				Quantity: 2,
			}},
		}
		start := make(chan struct{})
		results := make(chan reservationResult, 2)
		var waitGroup sync.WaitGroup
		for range 2 {
			waitGroup.Add(1)
			go func() {
				defer waitGroup.Done()
				<-start
				reservation, createErr := service.CreateReservation(
					t.Context(),
					input,
					actorID,
				)
				results <- reservationResult{reservation: reservation, err: createErr}
			}()
		}
		close(start)
		waitGroup.Wait()
		close(results)

		var reservationID uuid.UUID
		for result := range results {
			if result.err != nil {
				t.Fatal(result.err)
			}
			if reservationID == uuid.Nil {
				reservationID = result.reservation.ID
			} else if result.reservation.ID != reservationID {
				t.Fatalf("expected the same reservation, got %s and %s", reservationID, result.reservation.ID)
			}
		}
		stock, err := service.GetStock(t.Context(), warehouse.ID, sku.ID)
		if err != nil {
			t.Fatal(err)
		}
		if stock.Reserved != 2 || stock.Available != 3 {
			t.Fatalf("duplicate request reserved stock twice: %+v", stock)
		}
	})
}

type reservationResult struct {
	reservation domain.Reservation
	err         error
}

func cleanupFixture(t *testing.T, db *gorm.DB, warehouseID, productID uuid.UUID) {
	t.Helper()
	statements := []struct {
		query string
		args  []any
	}{
		{query: `
		DELETE FROM catalog.stock_reservation_items
		WHERE reservation_id IN (
			SELECT id FROM catalog.stock_reservations WHERE warehouse_id = ?
		)`, args: []any{warehouseID}},
		{query: "DELETE FROM catalog.stock_reservations WHERE warehouse_id = ?", args: []any{warehouseID}},
		{query: "DELETE FROM catalog.stock_movements WHERE warehouse_id = ?", args: []any{warehouseID}},
		{query: "DELETE FROM catalog.inventory_stocks WHERE warehouse_id = ?", args: []any{warehouseID}},
		{query: "DELETE FROM catalog.warehouses WHERE id = ?", args: []any{warehouseID}},
		{
			query: "DELETE FROM catalog.prices WHERE sku_id IN (SELECT id FROM catalog.skus WHERE product_id = ?)",
			args:  []any{productID},
		},
		{query: "DELETE FROM catalog.product_categories WHERE product_id = ?", args: []any{productID}},
		{query: "DELETE FROM catalog.product_images WHERE product_id = ?", args: []any{productID}},
		{query: "DELETE FROM catalog.skus WHERE product_id = ?", args: []any{productID}},
		{query: "DELETE FROM catalog.products WHERE id = ?", args: []any{productID}},
	}
	for _, statement := range statements {
		if err := db.Exec(statement.query, statement.args...).Error; err != nil {
			t.Errorf("cleanup fixture: %v", err)
		}
	}
}
