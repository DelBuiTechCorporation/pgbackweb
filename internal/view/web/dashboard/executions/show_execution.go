package executions

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path/filepath"

	"github.com/eduardolat/pgbackweb/internal/database/dbgen"
	"github.com/eduardolat/pgbackweb/internal/util/pathutil"
	"github.com/eduardolat/pgbackweb/internal/util/timeutil"
	"github.com/eduardolat/pgbackweb/internal/view/web/component"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	lucide "github.com/nodxdev/nodxgo-lucide"
)

func (h *handlers) downloadExecutionHandler(c echo.Context) error {
	ctx := c.Request().Context()

	executionID, err := uuid.Parse(c.Param("executionID"))
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	isLocal, links, err := h.servs.ExecutionsService.GetAllExecutionLinksOrPaths(
		ctx, executionID,
	)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	if len(links) == 0 {
		return c.String(http.StatusNotFound, "no files found")
	}

	// Single part: serve directly without repackaging
	if len(links) == 1 {
		if isLocal {
			return c.Attachment(links[0], filepath.Base(links[0]))
		}
		return c.Redirect(http.StatusFound, links[0])
	}

	// Multi-part: combine all SQL files from each zip part into a single zip
	filename := fmt.Sprintf("dump-%s.zip", executionID.String())
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, filename))
	c.Response().Header().Set("Content-Type", "application/zip")
	c.Response().WriteHeader(http.StatusOK)

	zw := zip.NewWriter(c.Response().Writer)
	defer zw.Close()

	for i, link := range links {
		var zr *zip.Reader

		if isLocal {
			rc, openErr := zip.OpenReader(link)
			if openErr != nil {
				return openErr
			}
			defer rc.Close()
			zr = &rc.Reader
		} else {
			// Download S3 part via pre-signed URL into memory, then open as zip
			resp, getErr := http.Get(link) //nolint:noctx
			if getErr != nil {
				return getErr
			}
			data, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if readErr != nil {
				return readErr
			}
			zr, err = zip.NewReader(bytes.NewReader(data), int64(len(data)))
			if err != nil {
				return err
			}
		}

		for _, f := range zr.File {
			ext := filepath.Ext(f.Name)
			base := f.Name[:len(f.Name)-len(ext)]
			entryName := fmt.Sprintf("%s-part%03d%s", base, i+1, ext)

			fw, createErr := zw.Create(entryName)
			if createErr != nil {
				return createErr
			}

			rc2, openErr := f.Open()
			if openErr != nil {
				return openErr
			}
			_, copyErr := io.Copy(fw, rc2)
			rc2.Close()
			if copyErr != nil {
				return copyErr
			}
		}
	}

	return zw.Close()
}

func showExecutionButton(
	execution dbgen.ExecutionsServicePaginateExecutionsRow,
) nodx.Node {
	mo := component.Modal(component.ModalParams{
		Title: "Execution details",
		Size:  component.SizeMd,
		Content: []nodx.Node{
			nodx.Div(
				nodx.Class("overflow-x-auto"),
				nodx.Table(
					nodx.Class("table [&_th]:text-nowrap"),
					nodx.Tr(
						nodx.Th(component.SpanText("ID")),
						nodx.Td(component.SpanText(execution.ID.String())),
					),
					nodx.Tr(
						nodx.Th(component.SpanText("Status")),
						nodx.Td(component.StatusBadge(execution.Status)),
					),
					nodx.Tr(
						nodx.Th(component.SpanText("Database")),
						nodx.Td(component.SpanText(execution.DatabaseName)),
					),
					nodx.Tr(
						nodx.Th(component.SpanText("Destination")),
						nodx.Td(component.PrettyDestinationName(
							execution.BackupIsLocal, execution.DestinationName,
						)),
					),
					nodx.If(
						execution.Message.Valid,
						nodx.Tr(
							nodx.Th(component.SpanText("Message")),
							nodx.Td(
								nodx.Class("break-all"),
								component.SpanText(execution.Message.String),
							),
						),
					),
					nodx.Tr(
						nodx.Th(component.SpanText("Started at")),
						nodx.Td(component.SpanText(
							execution.StartedAt.Local().Format(timeutil.LayoutYYYYMMDDHHMMSSPretty),
						)),
					),
					nodx.If(
						execution.FinishedAt.Valid,
						nodx.Tr(
							nodx.Th(component.SpanText("Finished at")),
							nodx.Td(component.SpanText(
								execution.FinishedAt.Time.Local().Format(timeutil.LayoutYYYYMMDDHHMMSSPretty),
							)),
						),
					),
					nodx.If(
						execution.FinishedAt.Valid,
						nodx.Tr(
							nodx.Th(component.SpanText("Took")),
							nodx.Td(component.SpanText(
								execution.FinishedAt.Time.Sub(execution.StartedAt).String(),
							)),
						),
					),
					nodx.If(
						execution.DeletedAt.Valid,
						nodx.Tr(
							nodx.Th(component.SpanText("Deleted at")),
							nodx.Td(component.SpanText(
								execution.DeletedAt.Time.Local().Format(timeutil.LayoutYYYYMMDDHHMMSSPretty),
							)),
						),
					),
					nodx.If(
						execution.FileSize.Valid,
						nodx.Tr(
							nodx.Th(component.SpanText("File size")),
							nodx.Td(component.PrettyFileSize(execution.FileSize)),
						),
					),
				),
				nodx.If(
					execution.Status == "success",
					nodx.Div(
						nodx.Class("flex justify-end items-center space-x-2"),
						deleteExecutionButton(execution.ID),
						buildDownloadButtons(execution),
					),
				),
			),
		},
	})

	return nodx.Div(
		mo.HTML,
		component.OptionsDropdownButton(
			mo.OpenerAttr,
			lucide.Eye(),
			component.SpanText("Show details"),
		),
	)
}

func buildDownloadButtons(execution dbgen.ExecutionsServicePaginateExecutionsRow) nodx.Node {
	return nodx.A(
		nodx.Href(pathutil.BuildPath(fmt.Sprintf("/dashboard/executions/%s/download", execution.ID))),
		nodx.Target("_blank"),
		nodx.Class("btn btn-primary"),
		component.SpanText("Download"),
		lucide.Download(),
	)
}
