package httpadapter

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/saaof/order-platform/catalog-service/internal/adapter/auth"
)

const authUserContextKey = "authUser"

func Authenticate(client *auth.Client) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(context echo.Context) error {
			header := strings.TrimSpace(context.Request().Header.Get("Authorization"))
			if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
				return echo.NewHTTPError(http.StatusUnauthorized, "Bearer token is required")
			}
			token := strings.TrimSpace(header[len("Bearer "):])
			user, err := client.Validate(context.Request().Context(), token)
			if err != nil {
				if errors.Is(err, auth.ErrUnavailable) {
					return echo.NewHTTPError(http.StatusServiceUnavailable, "Auth service is unavailable")
				}
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid access token")
			}
			context.Set(authUserContextKey, user)
			return next(context)
		}
	}
}

func RequirePermission(permission string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(context echo.Context) error {
			user, ok := context.Get(authUserContextKey).(auth.User)
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "Authentication is required")
			}
			if !user.HasPermission(permission) {
				return echo.NewHTTPError(http.StatusForbidden, "Insufficient permission")
			}
			return next(context)
		}
	}
}

func ActorID(context echo.Context) (uuid.UUID, error) {
	user, ok := context.Get(authUserContextKey).(auth.User)
	if !ok {
		return uuid.Nil, echo.NewHTTPError(http.StatusUnauthorized, "Authentication is required")
	}
	id, err := uuid.Parse(user.Subject)
	if err != nil {
		return uuid.Nil, echo.NewHTTPError(http.StatusUnauthorized, "Invalid authenticated user")
	}
	return id, nil
}
