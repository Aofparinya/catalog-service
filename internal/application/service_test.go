package application

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/saaof/order-platform/catalog-service/internal/domain"
)

func TestNormalizeAttributes(t *testing.T) {
	value, err := normalizeAttributes(json.RawMessage(`{"size":"XL","color":"black"}`))
	if err != nil {
		t.Fatalf("normalizeAttributes() error = %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(value, &result); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if result["size"] != "XL" {
		t.Fatalf("unexpected attributes: %v", result)
	}
	if _, err := normalizeAttributes(json.RawMessage(`["not","object"]`)); err == nil {
		t.Fatal("expected array attributes to fail")
	}
}

func TestCreateReservationValidation(t *testing.T) {
	service := NewService(nil)
	_, err := service.CreateReservation(t.Context(), CreateReservationInput{
		WarehouseID:   uuid.New(),
		ReferenceType: "ORDER",
		ReferenceID:   "order-1",
		ExpiresAt:     time.Now().Add(time.Minute),
		Items:         []domain.ReservationItem{{SKUID: uuid.New(), Quantity: 0}},
	}, uuid.New())
	if err == nil {
		t.Fatal("expected invalid quantity error")
	}
}
