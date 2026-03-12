package executions

import (
	"net/http"

	"github.com/eduardolat/pgbackweb/internal/util/echoutil"
	"github.com/eduardolat/pgbackweb/internal/util/pathutil"
	"github.com/eduardolat/pgbackweb/internal/util/strutil"
	"github.com/eduardolat/pgbackweb/internal/validate"
	"github.com/eduardolat/pgbackweb/internal/view/reqctx"
	"github.com/eduardolat/pgbackweb/internal/view/web/component"
	"github.com/eduardolat/pgbackweb/internal/view/web/layout"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	alpine "github.com/nodxdev/nodxgo-alpine"
	nodx "github.com/nodxdev/nodxgo"
	htmx "github.com/nodxdev/nodxgo-htmx"
)

type execsQueryData struct {
	Database    uuid.UUID `query:"database" validate:"omitempty,uuid"`
	Destination uuid.UUID `query:"destination" validate:"omitempty,uuid"`
	Backup      uuid.UUID `query:"backup" validate:"omitempty,uuid"`
	GroupBy     string    `query:"group_by" validate:"omitempty,oneof=day month year backup"`
}

func (h *handlers) indexPageHandler(c echo.Context) error {
	reqCtx := reqctx.GetCtx(c)

	var queryData execsQueryData
	if err := c.Bind(&queryData); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	if err := validate.Struct(&queryData); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	return echoutil.RenderNodx(c, http.StatusOK, indexPage(reqCtx, queryData))
}

func indexPage(reqCtx reqctx.Ctx, queryData execsQueryData) nodx.Node {
	groupByOptions := []struct{ Value, Label string }{
		{"", "No grouping"},
		{"day", "Day"},
		{"month", "Month"},
		{"year", "Year"},
		{"backup", "Backup task"},
	}

	content := []nodx.Node{
		component.H1Text("Executions"),
		nodx.Div(
			nodx.Class("mt-4 flex items-center gap-3"),
			component.SpanText("Group by:"),
			nodx.Select(
				nodx.Class("select select-sm select-bordered"),
				nodx.Attr("onchange", `var p=new URLSearchParams(window.location.search);p.set('group_by',this.value);window.location.href=window.location.pathname+'?'+p.toString();`),
				nodx.Map(groupByOptions, func(opt struct{ Value, Label string }) nodx.Node {
					return nodx.Option(
						nodx.Value(opt.Value),
						nodx.Text(opt.Label),
						nodx.If(queryData.GroupBy == opt.Value, nodx.Selected("")),
					)
				}),
			),
		),
		component.CardBox(component.CardBoxParams{
			Class: "mt-4",
			Children: []nodx.Node{
				nodx.Div(
					nodx.Class("overflow-x-auto"),
					nodx.Table(
						nodx.Class("table text-nowrap"),					nodx.If(queryData.GroupBy != "", alpine.XData("{groups:{}}")),
						nodx.Thead(
							nodx.Tr(
								nodx.Th(nodx.Class("w-1")),
								nodx.Th(component.SpanText("Status")),
								nodx.Th(component.SpanText("Backup")),
								nodx.Th(component.SpanText("Database")),
								nodx.Th(component.SpanText("Destination")),
								nodx.Th(component.SpanText("Started at")),
								nodx.Th(component.SpanText("Finished at")),
								nodx.Th(component.SpanText("Duration")),
								nodx.Th(component.SpanText("File size")),
							),
						),
						nodx.Tbody(
							component.SkeletonTr(8),
							htmx.HxGet(func() string {
								url := pathutil.BuildPath("/dashboard/executions/list?page=1")
								if queryData.Database != uuid.Nil {
									url = strutil.AddQueryParamToUrl(url, "database", queryData.Database.String())
								}
								if queryData.Destination != uuid.Nil {
									url = strutil.AddQueryParamToUrl(url, "destination", queryData.Destination.String())
								}
								if queryData.Backup != uuid.Nil {
									url = strutil.AddQueryParamToUrl(url, "backup", queryData.Backup.String())
								}
								if queryData.GroupBy != "" {
									url = strutil.AddQueryParamToUrl(url, "group_by", queryData.GroupBy)
								}
								return url
							}()),
							htmx.HxTrigger("load"),
						),
					),
				),
			},
		}),
	}

	return layout.Dashboard(reqCtx, layout.DashboardParams{
		Title: "Executions",
		Body:  content,
	})
}
