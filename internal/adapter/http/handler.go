package httpadapter

import (
	stdcontext "context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/saaof/order-platform/catalog-service/internal/application"
	"github.com/saaof/order-platform/catalog-service/internal/domain"
)

type Handler struct {
	service  *application.Service
	validate *validator.Validate
}

func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service, validate: validator.New()}
}

func (handler *Handler) Health(context echo.Context) error {
	return context.JSON(http.StatusOK, map[string]any{
		"status":  "ok",
		"service": "catalog-service",
	})
}

func (handler *Handler) CreateCategory(context echo.Context) error {
	var input application.CreateCategoryInput
	if err := handler.decodeAndValidate(context, &input); err != nil {
		return err
	}
	item, err := handler.service.CreateCategory(context.Request().Context(), input)
	if err != nil {
		return err
	}
	return context.JSON(http.StatusCreated, item)
}

func (handler *Handler) ListCategories(context echo.Context) error {
	items, err := handler.service.ListCategories(context.Request().Context())
	if err != nil {
		return err
	}
	return context.JSON(http.StatusOK, items)
}

func (handler *Handler) GetCategory(context echo.Context) error {
	id, err := pathUUID(context, "id")
	if err != nil {
		return err
	}
	item, err := handler.service.GetCategory(context.Request().Context(), id)
	if err != nil {
		return err
	}
	return context.JSON(http.StatusOK, item)
}

func (handler *Handler) UpdateCategory(context echo.Context) error {
	id, err := pathUUID(context, "id")
	if err != nil {
		return err
	}
	var input application.UpdateCategoryInput
	if err := handler.decodeAndValidate(context, &input); err != nil {
		return err
	}
	item, err := handler.service.UpdateCategory(context.Request().Context(), id, input)
	if err != nil {
		return err
	}
	return context.JSON(http.StatusOK, item)
}

func (handler *Handler) DeleteCategory(context echo.Context) error {
	id, err := pathUUID(context, "id")
	if err != nil {
		return err
	}
	if err := handler.service.DeleteCategory(context.Request().Context(), id); err != nil {
		return err
	}
	return context.NoContent(http.StatusNoContent)
}

func (handler *Handler) CreateProduct(context echo.Context) error {
	var input application.CreateProductInput
	if err := handler.decodeAndValidate(context, &input); err != nil {
		return err
	}
	actorID, err := ActorID(context)
	if err != nil {
		return err
	}
	item, err := handler.service.CreateProduct(context.Request().Context(), input, actorID)
	if err != nil {
		return err
	}
	return context.JSON(http.StatusCreated, item)
}

func (handler *Handler) ListProducts(context echo.Context) error {
	filter, err := productListFilter(context)
	if err != nil {
		return err
	}
	items, err := handler.service.ListProducts(context.Request().Context(), filter)
	if err != nil {
		return err
	}
	return context.JSON(http.StatusOK, items)
}

func (handler *Handler) GetProduct(context echo.Context) error {
	id, err := pathUUID(context, "id")
	if err != nil {
		return err
	}
	item, err := handler.service.GetProduct(context.Request().Context(), id)
	if err != nil {
		return err
	}
	return context.JSON(http.StatusOK, item)
}

func (handler *Handler) UpdateProduct(context echo.Context) error {
	id, err := pathUUID(context, "id")
	if err != nil {
		return err
	}
	var input application.UpdateProductInput
	if err := handler.decodeAndValidate(context, &input); err != nil {
		return err
	}
	actorID, err := ActorID(context)
	if err != nil {
		return err
	}
	item, err := handler.service.UpdateProduct(context.Request().Context(), id, input, actorID)
	if err != nil {
		return err
	}
	return context.JSON(http.StatusOK, item)
}

func (handler *Handler) DeleteProduct(context echo.Context) error {
	id, err := pathUUID(context, "id")
	if err != nil {
		return err
	}
	actorID, err := ActorID(context)
	if err != nil {
		return err
	}
	if err := handler.service.DeleteProduct(context.Request().Context(), id, actorID); err != nil {
		return err
	}
	return context.NoContent(http.StatusNoContent)
}

func (handler *Handler) AssignProductCategory(context echo.Context) error {
	productID, err := pathUUID(context, "id")
	if err != nil {
		return err
	}
	var input application.AssignCategoryInput
	if err := handler.decodeAndValidate(context, &input); err != nil {
		return err
	}
	item, err := handler.service.AssignCategory(context.Request().Context(), productID, input)
	if err != nil {
		return err
	}
	return context.JSON(http.StatusCreated, item)
}

func (handler *Handler) ListProductCategories(context echo.Context) error {
	productID, err := pathUUID(context, "id")
	if err != nil {
		return err
	}
	items, err := handler.service.ListProductCategories(context.Request().Context(), productID)
	if err != nil {
		return err
	}
	return context.JSON(http.StatusOK, items)
}

func (handler *Handler) RemoveProductCategory(context echo.Context) error {
	productID, categoryID, err := pathPair(context, "id", "categoryId")
	if err != nil {
		return err
	}
	if err := handler.service.RemoveCategory(context.Request().Context(), productID, categoryID); err != nil {
		return err
	}
	return context.NoContent(http.StatusNoContent)
}

func (handler *Handler) CreateProductImage(context echo.Context) error {
	productID, err := pathUUID(context, "id")
	if err != nil {
		return err
	}
	var input application.CreateImageInput
	if err := handler.decodeAndValidate(context, &input); err != nil {
		return err
	}
	item, err := handler.service.CreateImage(context.Request().Context(), productID, input)
	if err != nil {
		return err
	}
	return context.JSON(http.StatusCreated, item)
}

func (handler *Handler) ListProductImages(context echo.Context) error {
	productID, err := pathUUID(context, "id")
	if err != nil {
		return err
	}
	items, err := handler.service.ListImages(context.Request().Context(), productID)
	if err != nil {
		return err
	}
	return context.JSON(http.StatusOK, items)
}

func (handler *Handler) UpdateProductImage(context echo.Context) error {
	productID, imageID, err := pathPair(context, "id", "imageId")
	if err != nil {
		return err
	}
	var input application.UpdateImageInput
	if err := handler.decodeAndValidate(context, &input); err != nil {
		return err
	}
	item, err := handler.service.UpdateImage(context.Request().Context(), productID, imageID, input)
	if err != nil {
		return err
	}
	return context.JSON(http.StatusOK, item)
}

func (handler *Handler) DeleteProductImage(context echo.Context) error {
	productID, imageID, err := pathPair(context, "id", "imageId")
	if err != nil {
		return err
	}
	if err := handler.service.DeleteImage(context.Request().Context(), productID, imageID); err != nil {
		return err
	}
	return context.NoContent(http.StatusNoContent)
}

func (handler *Handler) CreateSKU(context echo.Context) error {
	productID, err := pathUUID(context, "id")
	if err != nil {
		return err
	}
	var input application.CreateSKUInput
	if err := handler.decodeAndValidate(context, &input); err != nil {
		return err
	}
	item, err := handler.service.CreateSKU(context.Request().Context(), productID, input)
	if err != nil {
		return err
	}
	return context.JSON(http.StatusCreated, item)
}

func (handler *Handler) ListSKUs(context echo.Context) error {
	productID, err := pathUUID(context, "id")
	if err != nil {
		return err
	}
	items, err := handler.service.ListSKUs(context.Request().Context(), productID)
	if err != nil {
		return err
	}
	return context.JSON(http.StatusOK, items)
}

func (handler *Handler) GetSKU(context echo.Context) error {
	id, err := pathUUID(context, "id")
	if err != nil {
		return err
	}
	item, err := handler.service.GetSKU(context.Request().Context(), id)
	if err != nil {
		return err
	}
	return context.JSON(http.StatusOK, item)
}

func (handler *Handler) UpdateSKU(context echo.Context) error {
	id, err := pathUUID(context, "id")
	if err != nil {
		return err
	}
	var input application.UpdateSKUInput
	if err := handler.decodeAndValidate(context, &input); err != nil {
		return err
	}
	item, err := handler.service.UpdateSKU(context.Request().Context(), id, input)
	if err != nil {
		return err
	}
	return context.JSON(http.StatusOK, item)
}

func (handler *Handler) DeleteSKU(context echo.Context) error {
	id, err := pathUUID(context, "id")
	if err != nil {
		return err
	}
	if err := handler.service.DeleteSKU(context.Request().Context(), id); err != nil {
		return err
	}
	return context.NoContent(http.StatusNoContent)
}

func (handler *Handler) CreatePrice(context echo.Context) error {
	skuID, err := pathUUID(context, "id")
	if err != nil {
		return err
	}
	var input application.CreatePriceInput
	if err := handler.decodeAndValidate(context, &input); err != nil {
		return err
	}
	item, err := handler.service.CreatePrice(context.Request().Context(), skuID, input)
	if err != nil {
		return err
	}
	return context.JSON(http.StatusCreated, item)
}

func (handler *Handler) ListPrices(context echo.Context) error {
	skuID, err := pathUUID(context, "id")
	if err != nil {
		return err
	}
	items, err := handler.service.ListPrices(context.Request().Context(), skuID)
	if err != nil {
		return err
	}
	return context.JSON(http.StatusOK, items)
}

func (handler *Handler) UpdatePrice(context echo.Context) error {
	skuID, priceID, err := pathPair(context, "id", "priceId")
	if err != nil {
		return err
	}
	var input application.UpdatePriceInput
	if err := handler.decodeAndValidate(context, &input); err != nil {
		return err
	}
	item, err := handler.service.UpdatePrice(context.Request().Context(), skuID, priceID, input)
	if err != nil {
		return err
	}
	return context.JSON(http.StatusOK, item)
}

func (handler *Handler) DeletePrice(context echo.Context) error {
	skuID, priceID, err := pathPair(context, "id", "priceId")
	if err != nil {
		return err
	}
	if err := handler.service.DeletePrice(context.Request().Context(), skuID, priceID); err != nil {
		return err
	}
	return context.NoContent(http.StatusNoContent)
}

func (handler *Handler) CreateWarehouse(context echo.Context) error {
	var input application.CreateWarehouseInput
	if err := handler.decodeAndValidate(context, &input); err != nil {
		return err
	}
	item, err := handler.service.CreateWarehouse(context.Request().Context(), input)
	if err != nil {
		return err
	}
	return context.JSON(http.StatusCreated, item)
}

func (handler *Handler) ListWarehouses(context echo.Context) error {
	items, err := handler.service.ListWarehouses(context.Request().Context())
	if err != nil {
		return err
	}
	return context.JSON(http.StatusOK, items)
}

func (handler *Handler) UpdateWarehouse(context echo.Context) error {
	id, err := pathUUID(context, "id")
	if err != nil {
		return err
	}
	var input application.UpdateWarehouseInput
	if err := handler.decodeAndValidate(context, &input); err != nil {
		return err
	}
	item, err := handler.service.UpdateWarehouse(context.Request().Context(), id, input)
	if err != nil {
		return err
	}
	return context.JSON(http.StatusOK, item)
}

func (handler *Handler) DeleteWarehouse(context echo.Context) error {
	id, err := pathUUID(context, "id")
	if err != nil {
		return err
	}
	if err := handler.service.DeleteWarehouse(context.Request().Context(), id); err != nil {
		return err
	}
	return context.NoContent(http.StatusNoContent)
}

func (handler *Handler) ListInventory(context echo.Context) error {
	filter, err := inventoryListFilter(context)
	if err != nil {
		return err
	}
	items, err := handler.service.ListInventory(context.Request().Context(), filter)
	if err != nil {
		return err
	}
	return context.JSON(http.StatusOK, items)
}

func (handler *Handler) GetStock(context echo.Context) error {
	warehouseID, skuID, err := pathPair(context, "warehouseId", "skuId")
	if err != nil {
		return err
	}
	item, err := handler.service.GetStock(context.Request().Context(), warehouseID, skuID)
	if err != nil {
		return err
	}
	return context.JSON(http.StatusOK, item)
}

func (handler *Handler) AdjustStock(context echo.Context) error {
	var input application.AdjustmentInput
	if err := handler.decodeAndValidate(context, &input); err != nil {
		return err
	}
	actorID, err := ActorID(context)
	if err != nil {
		return err
	}
	item, err := handler.service.AdjustStock(context.Request().Context(), input, actorID)
	if err != nil {
		return err
	}
	return context.JSON(http.StatusOK, item)
}

func (handler *Handler) ListMovements(context echo.Context) error {
	filter, err := inventoryListFilter(context)
	if err != nil {
		return err
	}
	items, err := handler.service.ListMovements(context.Request().Context(), filter)
	if err != nil {
		return err
	}
	return context.JSON(http.StatusOK, items)
}

func (handler *Handler) CreateReservation(context echo.Context) error {
	var input application.CreateReservationInput
	if err := handler.decodeAndValidate(context, &input); err != nil {
		return err
	}
	actorID, err := ActorID(context)
	if err != nil {
		return err
	}
	item, err := handler.service.CreateReservation(context.Request().Context(), input, actorID)
	if err != nil {
		return err
	}
	return context.JSON(http.StatusCreated, item)
}

func (handler *Handler) ListReservations(context echo.Context) error {
	filter, err := reservationListFilter(context)
	if err != nil {
		return err
	}
	items, err := handler.service.ListReservations(context.Request().Context(), filter)
	if err != nil {
		return err
	}
	return context.JSON(http.StatusOK, items)
}

func (handler *Handler) GetReservation(context echo.Context) error {
	id, err := pathUUID(context, "id")
	if err != nil {
		return err
	}
	item, err := handler.service.GetReservation(context.Request().Context(), id)
	if err != nil {
		return err
	}
	return context.JSON(http.StatusOK, item)
}

func (handler *Handler) ConfirmReservation(context echo.Context) error {
	return handler.transitionReservation(context, handler.service.ConfirmReservation)
}

func (handler *Handler) ReleaseReservation(context echo.Context) error {
	return handler.transitionReservation(context, handler.service.ReleaseReservation)
}

func (handler *Handler) transitionReservation(
	context echo.Context,
	transition func(ctx stdcontext.Context, id uuid.UUID, actorID uuid.UUID) (domain.Reservation, error),
) error {
	id, err := pathUUID(context, "id")
	if err != nil {
		return err
	}
	actorID, err := ActorID(context)
	if err != nil {
		return err
	}
	item, err := transition(context.Request().Context(), id, actorID)
	if err != nil {
		return err
	}
	return context.JSON(http.StatusOK, item)
}

func (handler *Handler) decodeAndValidate(context echo.Context, target any) error {
	decoder := json.NewDecoder(context.Request().Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		if errors.Is(err, io.EOF) {
			return echo.NewHTTPError(http.StatusBadRequest, "Request body is required")
		}
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
	}
	if err := handler.validate.Struct(target); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, validationMessage(err))
	}
	return nil
}

func productListFilter(context echo.Context) (application.ListFilter, error) {
	filter, err := baseListFilter(context)
	if err != nil {
		return application.ListFilter{}, err
	}
	if value := strings.TrimSpace(context.QueryParam("status")); value != "" {
		status := strings.ToUpper(value)
		if status != string(domain.ProductStatusDraft) &&
			status != string(domain.ProductStatusActive) &&
			status != string(domain.ProductStatusInactive) {
			return application.ListFilter{}, echo.NewHTTPError(http.StatusBadRequest, "invalid status")
		}
		filter.Status = status
	}
	if value := strings.TrimSpace(context.QueryParam("categoryId")); value != "" {
		id, err := uuid.Parse(value)
		if err != nil {
			return application.ListFilter{}, echo.NewHTTPError(http.StatusBadRequest, "categoryId must be a valid UUID")
		}
		filter.CategoryID = &id
	}
	return filter, nil
}

func inventoryListFilter(context echo.Context) (application.ListFilter, error) {
	filter, err := baseListFilter(context)
	if err != nil {
		return application.ListFilter{}, err
	}
	if value := strings.TrimSpace(context.QueryParam("warehouseId")); value != "" {
		id, err := uuid.Parse(value)
		if err != nil {
			return application.ListFilter{}, echo.NewHTTPError(http.StatusBadRequest, "warehouseId must be a valid UUID")
		}
		filter.WarehouseID = &id
	}
	if value := strings.TrimSpace(context.QueryParam("skuId")); value != "" {
		id, err := uuid.Parse(value)
		if err != nil {
			return application.ListFilter{}, echo.NewHTTPError(http.StatusBadRequest, "skuId must be a valid UUID")
		}
		filter.SKUID = &id
	}
	if value := strings.TrimSpace(context.QueryParam("lowStock")); value != "" {
		lowStock, err := strconv.ParseBool(value)
		if err != nil {
			return application.ListFilter{}, echo.NewHTTPError(http.StatusBadRequest, "lowStock must be true or false")
		}
		filter.LowStock = lowStock
	}
	return filter, nil
}

func reservationListFilter(context echo.Context) (application.ListFilter, error) {
	filter, err := baseListFilter(context)
	if err != nil {
		return application.ListFilter{}, err
	}
	if value := strings.TrimSpace(context.QueryParam("warehouseId")); value != "" {
		id, err := uuid.Parse(value)
		if err != nil {
			return application.ListFilter{}, echo.NewHTTPError(http.StatusBadRequest, "warehouseId must be a valid UUID")
		}
		filter.WarehouseID = &id
	}
	if value := strings.ToUpper(strings.TrimSpace(context.QueryParam("status"))); value != "" {
		if value != string(domain.ReservationPending) &&
			value != string(domain.ReservationConfirmed) &&
			value != string(domain.ReservationReleased) &&
			value != string(domain.ReservationExpired) {
			return application.ListFilter{}, echo.NewHTTPError(http.StatusBadRequest, "invalid status")
		}
		filter.Status = value
	}
	filter.ReferenceType = context.QueryParam("referenceType")
	filter.ReferenceID = context.QueryParam("referenceId")
	return filter, nil
}

func baseListFilter(context echo.Context) (application.ListFilter, error) {
	page, err := positiveQueryInt(context.QueryParam("page"), 1)
	if err != nil {
		return application.ListFilter{}, echo.NewHTTPError(http.StatusBadRequest, "page must be a positive integer")
	}
	pageSize, err := positiveQueryInt(context.QueryParam("pageSize"), 20)
	if err != nil {
		return application.ListFilter{}, echo.NewHTTPError(http.StatusBadRequest, "pageSize must be a positive integer")
	}
	return application.ListFilter{
		Query:    context.QueryParam("q"),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func positiveQueryInt(value string, fallback int) (int, error) {
	if strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	result, err := strconv.Atoi(value)
	if err != nil || result <= 0 {
		return 0, errors.New("invalid positive integer")
	}
	return result, nil
}

func pathUUID(context echo.Context, name string) (uuid.UUID, error) {
	value, err := uuid.Parse(context.Param(name))
	if err != nil {
		return uuid.Nil, echo.NewHTTPError(http.StatusBadRequest, name+" must be a valid UUID")
	}
	return value, nil
}

func pathPair(context echo.Context, firstName, secondName string) (uuid.UUID, uuid.UUID, error) {
	firstID, err := pathUUID(context, firstName)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	secondID, err := pathUUID(context, secondName)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return firstID, secondID, nil
}

func validationMessage(err error) string {
	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) || len(validationErrors) == 0 {
		return "Request validation failed"
	}
	item := validationErrors[0]
	return fmt.Sprintf("%s failed validation rule %s", item.Field(), item.Tag())
}
