package httpadapter

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/saaof/order-platform/catalog-service/internal/adapter/auth"
	"github.com/saaof/order-platform/catalog-service/internal/domain"
)

func NewRouter(handler *Handler, authClient *auth.Client, corsOrigins []string) *echo.Echo {
	server := echo.New()
	server.HideBanner = true
	server.HTTPErrorHandler = errorHandler
	server.Use(middleware.Recover())
	server.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:      true,
		LogStatus:   true,
		LogMethod:   true,
		LogLatency:  true,
		LogError:    true,
		HandleError: true,
		LogValuesFunc: func(_ echo.Context, values middleware.RequestLoggerValues) error {
			slog.Info("http request",
				"method", values.Method,
				"uri", values.URI,
				"status", values.Status,
				"latency", values.Latency,
				"error", values.Error,
			)
			return nil
		},
	}))
	server.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: corsOrigins,
		AllowHeaders: []string{
			echo.HeaderOrigin,
			echo.HeaderContentType,
			echo.HeaderAccept,
			echo.HeaderAuthorization,
		},
	}))

	api := server.Group("/api/v1")
	api.GET("/health", handler.Health)

	protected := api.Group("", Authenticate(authClient))
	catalogRead := RequirePermission("catalog.read")
	catalogWrite := RequirePermission("catalog.write")
	inventoryRead := RequirePermission("inventory.read")
	inventoryWrite := RequirePermission("inventory.write")

	protected.POST("/categories", handler.CreateCategory, catalogWrite)
	protected.GET("/categories", handler.ListCategories, catalogRead)
	protected.GET("/categories/:id", handler.GetCategory, catalogRead)
	protected.PATCH("/categories/:id", handler.UpdateCategory, catalogWrite)
	protected.DELETE("/categories/:id", handler.DeleteCategory, catalogWrite)

	protected.POST("/products", handler.CreateProduct, catalogWrite)
	protected.GET("/products", handler.ListProducts, catalogRead)
	protected.GET("/products/:id", handler.GetProduct, catalogRead)
	protected.PATCH("/products/:id", handler.UpdateProduct, catalogWrite)
	protected.DELETE("/products/:id", handler.DeleteProduct, catalogWrite)
	protected.POST("/products/:id/categories", handler.AssignProductCategory, catalogWrite)
	protected.DELETE("/products/:id/categories/:categoryId", handler.RemoveProductCategory, catalogWrite)
	protected.POST("/products/:id/images", handler.CreateProductImage, catalogWrite)
	protected.PATCH("/products/:id/images/:imageId", handler.UpdateProductImage, catalogWrite)
	protected.DELETE("/products/:id/images/:imageId", handler.DeleteProductImage, catalogWrite)
	protected.POST("/products/:id/skus", handler.CreateSKU, catalogWrite)
	protected.GET("/products/:id/skus", handler.ListSKUs, catalogRead)

	protected.GET("/skus/:id", handler.GetSKU, catalogRead)
	protected.PATCH("/skus/:id", handler.UpdateSKU, catalogWrite)
	protected.DELETE("/skus/:id", handler.DeleteSKU, catalogWrite)
	protected.POST("/skus/:id/prices", handler.CreatePrice, catalogWrite)
	protected.GET("/skus/:id/prices", handler.ListPrices, catalogRead)
	protected.PATCH("/skus/:id/prices/:priceId", handler.UpdatePrice, catalogWrite)
	protected.DELETE("/skus/:id/prices/:priceId", handler.DeletePrice, catalogWrite)

	protected.POST("/warehouses", handler.CreateWarehouse, inventoryWrite)
	protected.GET("/warehouses", handler.ListWarehouses, inventoryRead)
	protected.PATCH("/warehouses/:id", handler.UpdateWarehouse, inventoryWrite)
	protected.DELETE("/warehouses/:id", handler.DeleteWarehouse, inventoryWrite)

	protected.GET("/inventory", handler.ListInventory, inventoryRead)
	protected.GET("/inventory/movements", handler.ListMovements, inventoryRead)
	protected.POST("/inventory/adjustments", handler.AdjustStock, inventoryWrite)
	protected.POST("/inventory/reservations", handler.CreateReservation, inventoryWrite)
	protected.GET("/inventory/reservations/:id", handler.GetReservation, inventoryRead)
	protected.POST("/inventory/reservations/:id/confirm", handler.ConfirmReservation, inventoryWrite)
	protected.POST("/inventory/reservations/:id/release", handler.ReleaseReservation, inventoryWrite)
	protected.GET("/inventory/:warehouseId/:skuId", handler.GetStock, inventoryRead)

	return server
}

func errorHandler(err error, context echo.Context) {
	if context.Response().Committed {
		return
	}
	status := http.StatusInternalServerError
	message := "Internal server error"
	errorName := "Internal Server Error"

	var httpError *echo.HTTPError
	switch {
	case errors.As(err, &httpError):
		status = httpError.Code
		message = messageFromHTTPError(httpError)
		errorName = http.StatusText(status)
	case errors.Is(err, domain.ErrNotFound):
		status = http.StatusNotFound
		message = "Resource not found"
		errorName = "Not Found"
	case errors.Is(err, domain.ErrConflict), errors.Is(err, domain.ErrInsufficientStock):
		status = http.StatusConflict
		message = err.Error()
		errorName = "Conflict"
	case errors.Is(err, domain.ErrInvalid):
		status = http.StatusBadRequest
		message = err.Error()
		errorName = "Bad Request"
	default:
		slog.Error("unhandled request error", "error", err)
	}

	_ = context.JSON(status, map[string]any{
		"message":    message,
		"error":      errorName,
		"statusCode": status,
	})
}

func messageFromHTTPError(err *echo.HTTPError) string {
	if message, ok := err.Message.(string); ok {
		return message
	}
	return http.StatusText(err.Code)
}
