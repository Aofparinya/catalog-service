package httpadapter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/saaof/order-platform/catalog-service/internal/adapter/auth"
)

func TestRequirePermission(t *testing.T) {
	server := echo.New()
	context := server.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
	context.Set(authUserContextKey, auth.User{Permissions: []string{"catalog.read"}})
	if err := RequirePermission("catalog.read")(func(echo.Context) error { return nil })(context); err != nil {
		t.Fatalf("expected permission: %v", err)
	}
	if err := RequirePermission("catalog.write")(func(echo.Context) error { return nil })(context); err == nil {
		t.Fatal("expected forbidden")
	}
}
