package middleware

import (
	"net/http"

	"github.com/eduardolat/pgbackweb/internal/view/reqctx"
	"github.com/labstack/echo/v4"
	htmx "github.com/nodxdev/nodxgo-htmx"
)

func (m *Middleware) RequireNoAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		reqCtx := reqctx.GetCtx(c)

		if reqCtx.IsAuthed {
			htmx.ServerSetRedirect(c.Response().Header(), "/dashboard")
			return c.Redirect(http.StatusFound, "/dashboard")
		}

		return next(c)
	}
}
