package executions

import (
	"fmt"
	"net/http"

	"github.com/eduardolat/pgbackweb/internal/database/dbgen"
	"github.com/eduardolat/pgbackweb/internal/service/executions"
	"github.com/eduardolat/pgbackweb/internal/util/echoutil"
	"github.com/eduardolat/pgbackweb/internal/util/paginateutil"
	"github.com/eduardolat/pgbackweb/internal/util/pathutil"
	"github.com/eduardolat/pgbackweb/internal/util/strutil"
	"github.com/eduardolat/pgbackweb/internal/util/timeutil"
	"github.com/eduardolat/pgbackweb/internal/validate"
	"github.com/eduardolat/pgbackweb/internal/view/web/component"
	"github.com/eduardolat/pgbackweb/internal/view/web/respondhtmx"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	alpine "github.com/nodxdev/nodxgo-alpine"
	nodx "github.com/nodxdev/nodxgo"
	htmx "github.com/nodxdev/nodxgo-htmx"
	lucide "github.com/nodxdev/nodxgo-lucide"
)

type listExecsQueryData struct {
	Database     uuid.UUID `query:"database" validate:"omitempty,uuid"`
	Destination  uuid.UUID `query:"destination" validate:"omitempty,uuid"`
	Backup       uuid.UUID `query:"backup" validate:"omitempty,uuid"`
	Page         int       `query:"page" validate:"required,min=1"`
	GroupBy      string    `query:"group_by" validate:"omitempty,oneof=day month year backup"`
	LastGroupKey string    `query:"last_group_key"`
}

func (h *handlers) listExecutionsHandler(c echo.Context) error {
	ctx := c.Request().Context()

	var queryData listExecsQueryData
	if err := c.Bind(&queryData); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}
	if err := validate.Struct(&queryData); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	pagination, executions, err := h.servs.ExecutionsService.PaginateExecutions(
		ctx, executions.PaginateExecutionsParams{
			DatabaseFilter: uuid.NullUUID{
				UUID: queryData.Database, Valid: queryData.Database != uuid.Nil,
			},
			DestinationFilter: uuid.NullUUID{
				UUID: queryData.Destination, Valid: queryData.Destination != uuid.Nil,
			},
			BackupFilter: uuid.NullUUID{
				UUID: queryData.Backup, Valid: queryData.Backup != uuid.Nil,
			},
			Page:  queryData.Page,
			Limit: 20,
		},
	)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	return echoutil.RenderNodx(
		c, http.StatusOK, listExecutions(queryData, pagination, executions),
	)
}

func execGroupKey(execution dbgen.ExecutionsServicePaginateExecutionsRow, groupBy string) string {
	switch groupBy {
	case "day":
		return execution.StartedAt.Local().Format("2006-01-02")
	case "month":
		return execution.StartedAt.Local().Format("2006-01")
	case "year":
		return execution.StartedAt.Local().Format("2006")
	case "backup":
		return execution.BackupID.String()
	}
	return ""
}

func execGroupLabel(execution dbgen.ExecutionsServicePaginateExecutionsRow, groupBy string) string {
	switch groupBy {
	case "day":
		return execution.StartedAt.Local().Format("Monday, January 02, 2006")
	case "month":
		return execution.StartedAt.Local().Format("January 2006")
	case "year":
		return execution.StartedAt.Local().Format("2006")
	case "backup":
		return execution.BackupName
	}
	return ""
}

func execGroupHeaderTr(key, label string) nodx.Node {
	return nodx.Tr(
		nodx.Class("cursor-pointer select-none hover:opacity-80"),
		alpine.XOn("click", fmt.Sprintf("groups['%s']=!groups['%s']", key, key)),
		nodx.Td(
			nodx.Attr("colspan", "9"),
			nodx.Class("bg-base-300 border-y border-base-content/10 py-2 px-4"),
			nodx.Div(
				nodx.Class("flex items-center gap-2 font-semibold text-sm"),
				nodx.SpanEl(
					nodx.Class("inline-block"),
					alpine.XBind("style", fmt.Sprintf(
						`'transition:transform 0.2s;transform:' + (groups['%s'] ? 'rotate(-90deg)' : 'rotate(0deg)')`,
						key,
					)),
					lucide.ChevronDown(),
				),
				nodx.SpanEl(nodx.Text(label)),
			),
		),
	)
}

func listExecutions(
	queryData listExecsQueryData,
	pagination paginateutil.PaginateResponse,
	executions []dbgen.ExecutionsServicePaginateExecutionsRow,
) nodx.Node {
	if len(executions) < 1 {
		return component.EmptyResultsTr(component.EmptyResultsParams{
			Title:    "No executions found",
			Subtitle: "Wait for the first execution to appear here",
		})
	}

	currentGroup := queryData.LastGroupKey
	trs := []nodx.Node{}
	for _, execution := range executions {
		groupKey := execGroupKey(execution, queryData.GroupBy)
		if groupKey != "" && groupKey != currentGroup {
			trs = append(trs, execGroupHeaderTr(groupKey, execGroupLabel(execution, queryData.GroupBy)))
			currentGroup = groupKey
		}

		trs = append(trs, nodx.Tr(
			nodx.If(queryData.GroupBy != "" && groupKey != "", alpine.XShow(fmt.Sprintf("!groups['%s']", groupKey))),
			nodx.Td(component.OptionsDropdown(
				showExecutionButton(execution),
				restoreExecutionButton(execution),
			)),
			nodx.Td(component.StatusBadge(execution.Status)),
			nodx.Td(component.SpanText(execution.BackupName)),
			nodx.Td(component.SpanText(execution.DatabaseName)),
			nodx.Td(component.PrettyDestinationName(
				execution.BackupIsLocal, execution.DestinationName,
			)),
			nodx.Td(component.SpanText(
				execution.StartedAt.Local().Format(timeutil.LayoutYYYYMMDDHHMMSSPretty),
			)),
			nodx.Td(
				nodx.If(
					execution.FinishedAt.Valid,
					component.SpanText(
						execution.FinishedAt.Time.Local().Format(timeutil.LayoutYYYYMMDDHHMMSSPretty),
					),
				),
			),
			nodx.Td(
				nodx.If(
					execution.FinishedAt.Valid,
					component.SpanText(
						execution.FinishedAt.Time.Sub(execution.StartedAt).String(),
					),
				),
			),
			nodx.Td(
				nodx.If(
					execution.FileSize.Valid,
					component.PrettyFileSize(execution.FileSize),
				),
			),
		))
	}

	if pagination.HasNextPage {
		lastGroupKey := ""
		if len(executions) > 0 {
			lastGroupKey = execGroupKey(executions[len(executions)-1], queryData.GroupBy)
		}
		trs = append(trs, nodx.Tr(
			htmx.HxGet(func() string {
				url := pathutil.BuildPath("/dashboard/executions/list")
				url = strutil.AddQueryParamToUrl(url, "page", fmt.Sprintf("%d", pagination.NextPage))
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
					url = strutil.AddQueryParamToUrl(url, "last_group_key", lastGroupKey)
				}
				return url
			}()),
			htmx.HxTrigger("intersect once"),
			htmx.HxSwap("afterend"),
		))
	}

	return component.RenderableGroup(trs)
}
