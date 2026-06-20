package httpadapter

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/saaof/order-platform/catalog-service/internal/domain"
)

func TestHandlerRejectsMalformedRequests(t *testing.T) {
	server := echo.New()
	handler := NewHandler(nil)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/categories",
		strings.NewReader(`{"code":"TEST","unknown":true}`),
	)
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	context := server.NewContext(request, httptest.NewRecorder())
	if err := handler.CreateCategory(context); err == nil {
		t.Fatal("expected unknown field to be rejected")
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/categories/not-a-uuid", nil)
	context = server.NewContext(request, httptest.NewRecorder())
	context.SetPath("/api/v1/categories/:id")
	context.SetParamNames("id")
	context.SetParamValues("not-a-uuid")
	if err := handler.GetCategory(context); err == nil {
		t.Fatal("expected invalid UUID to be rejected")
	}
}

func TestErrorHandlerMapsDomainErrors(t *testing.T) {
	server := echo.New()
	recorder := httptest.NewRecorder()
	context := server.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), recorder)

	errorHandler(domain.ErrNotFound, context)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	context = server.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), recorder)
	errorHandler(domain.ErrInsufficientStock, context)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", recorder.Code)
	}
}
