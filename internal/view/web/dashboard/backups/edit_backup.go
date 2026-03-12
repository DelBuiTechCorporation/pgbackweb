package backups

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/eduardolat/pgbackweb/internal/database/dbgen"
	"github.com/eduardolat/pgbackweb/internal/staticdata"
	"github.com/eduardolat/pgbackweb/internal/util/echoutil"
	"github.com/eduardolat/pgbackweb/internal/util/pathutil"
	"github.com/eduardolat/pgbackweb/internal/validate"
	"github.com/eduardolat/pgbackweb/internal/view/web/component"
	"github.com/eduardolat/pgbackweb/internal/view/web/respondhtmx"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	alpine "github.com/nodxdev/nodxgo-alpine"
	htmx "github.com/nodxdev/nodxgo-htmx"
	lucide "github.com/nodxdev/nodxgo-lucide"
)

func (h *handlers) editBackupHandler(c echo.Context) error {
	ctx := c.Request().Context()

	backupID, err := uuid.Parse(c.Param("backupID"))
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	var formData struct {
		Name             string    `form:"name" validate:"required"`
		CronExpression   string    `form:"cron_expression" validate:"required"`
		TimeZone         string    `form:"time_zone" validate:"required"`
		IsActive         string    `form:"is_active" validate:"required,oneof=true false"`
		DestDir          string    `form:"dest_dir" validate:"required"`
		RetentionDays    int16     `form:"retention_days"`
		OptDataOnly      string    `form:"opt_data_only" validate:"required,oneof=true false"`
		OptSchemaOnly    string    `form:"opt_schema_only" validate:"required,oneof=true false"`
		OptClean         string    `form:"opt_clean" validate:"required,oneof=true false"`
		OptIfExists      string    `form:"opt_if_exists" validate:"required,oneof=true false"`
		OptCreate        string    `form:"opt_create" validate:"required,oneof=true false"`
		OptNoComments    string    `form:"opt_no_comments" validate:"required,oneof=true false"`
		IsLocal          string    `form:"is_local" validate:"required,oneof=true false"`
		DestinationID    uuid.UUID `form:"destination_id" validate:"omitempty,uuid"`
		MaxPartSizeMb    string    `form:"max_part_size_mb"`
		CompressionLevel string    `form:"compression_level"`
	}
	if err := c.Bind(&formData); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}
	if err := validate.Struct(&formData); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	_, err = h.servs.BackupsService.UpdateBackup(
		ctx, dbgen.BackupsServiceUpdateBackupParams{
			ID:             backupID,
			Name:           sql.NullString{String: formData.Name, Valid: true},
			CronExpression: sql.NullString{String: formData.CronExpression, Valid: true},
			TimeZone:       sql.NullString{String: formData.TimeZone, Valid: true},
			IsActive:       sql.NullBool{Bool: formData.IsActive == "true", Valid: true},
			DestDir:        sql.NullString{String: formData.DestDir, Valid: true},
			RetentionDays:  sql.NullInt16{Int16: formData.RetentionDays, Valid: true},
			OptDataOnly:    sql.NullBool{Bool: formData.OptDataOnly == "true", Valid: true},
			OptSchemaOnly:  sql.NullBool{Bool: formData.OptSchemaOnly == "true", Valid: true},
			OptClean:       sql.NullBool{Bool: formData.OptClean == "true", Valid: true},
			OptIfExists:    sql.NullBool{Bool: formData.OptIfExists == "true", Valid: true},
			OptCreate:      sql.NullBool{Bool: formData.OptCreate == "true", Valid: true},
			OptNoComments:  sql.NullBool{Bool: formData.OptNoComments == "true", Valid: true},
			IsLocal:        sql.NullBool{Bool: formData.IsLocal == "true", Valid: true},
			DestinationID: uuid.NullUUID{
				Valid: formData.IsLocal == "false", UUID: formData.DestinationID,
			},
			MaxPartSizeMb:    parseNullInt32(formData.MaxPartSizeMb),
			CompressionLevel: parseNullInt16(formData.CompressionLevel),
		},
	)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	return respondhtmx.AlertWithRefresh(c, "Backup task updated")
}

func (h *handlers) getEditBackupFormHandler(c echo.Context) error {
	ctx := c.Request().Context()

	backupID, err := uuid.Parse(c.Param("backupID"))
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	backup, err := h.servs.BackupsService.GetBackup(ctx, backupID)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	destinations, err := h.servs.DestinationsService.GetAllDestinations(ctx)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	return echoutil.RenderNodx(
		c, http.StatusOK, editBackupForm(backup, destinations),
	)
}

func editBackupButton(backup dbgen.BackupsServicePaginateBackupsRow) nodx.Node {
	mo := component.Modal(component.ModalParams{
		Size:  component.SizeLg,
		Title: "Edit backup task",
		Content: []nodx.Node{
			nodx.Div(
				htmx.HxGet(pathutil.BuildPath(
					fmt.Sprintf("/dashboard/backups/%s/edit-form", backup.ID),
				)),
				htmx.HxSwap("outerHTML"),
				htmx.HxTrigger("intersect once"),
				nodx.Class("p-10 flex justify-center"),
				component.HxLoadingMd(),
			),
		},
	})

	return nodx.Div(
		mo.HTML,
		component.OptionsDropdownButton(
			mo.OpenerAttr,
			lucide.Pencil(),
			component.SpanText("Edit backup task"),
		),
	)
}

func editBackupForm(
	backup dbgen.Backup,
	destinations []dbgen.DestinationsServiceGetAllDestinationsRow,
) nodx.Node {
	yesNoOptions := func(value bool) nodx.Node {
		return nodx.Group(
			nodx.Option(
				nodx.Value("true"),
				nodx.Text("Yes"),
				nodx.If(value, nodx.Selected("")),
			),
			nodx.Option(
				nodx.Value("false"),
				nodx.Text("No"),
				nodx.If(!value, nodx.Selected("")),
			),
		)
	}

	return nodx.FormEl(
		htmx.HxPost(pathutil.BuildPath(fmt.Sprintf("/dashboard/backups/%s/edit", backup.ID))),
		htmx.HxDisabledELT("find button"),
		nodx.Class("space-y-2 text-base"),

		alpine.XData(`{
					is_local: ` + fmt.Sprintf("%v", backup.IsLocal) + `,
				}`),

				component.InputControl(component.InputControlParams{
					Name:        "name",
					Label:       "Name",
					Placeholder: "My backup",
					Required:    true,
					Type:        component.InputTypeText,
					Children: []nodx.Node{
						nodx.Value(backup.Name),
					},
				}),

				component.InputControl(component.InputControlParams{
					Name:        "cron_expression",
					Label:       "Cron expression",
					Placeholder: "* * * * *",
					Required:    true,
					Type:        component.InputTypeText,
					HelpText:    "The cron expression to schedule the backup",
					Pattern:     `^\S+\s+\S+\s+\S+\s+\S+\s+\S+$`,
					Children: []nodx.Node{
						nodx.Value(backup.CronExpression),
					},
					HelpButtonChildren: cronExpressionHelp(),
				}),

				component.SelectControl(component.SelectControlParams{
					Name:        "time_zone",
					Label:       "Time zone",
					Required:    true,
					Placeholder: "Select a time zone",
					Children: []nodx.Node{
						nodx.Map(
							staticdata.Timezones,
							func(tz staticdata.Timezone) nodx.Node {
								return nodx.Option(
									nodx.Value(tz.TzCode),
									nodx.Text(tz.Label),
									nodx.If(
										tz.TzCode == backup.TimeZone,
										nodx.Selected(""),
									),
								)
							},
						),
					},
					HelpButtonChildren: timezoneFilenamesHelp(),
				}),

				component.InputControl(component.InputControlParams{
					Name:               "dest_dir",
					Label:              "Destination directory",
					Placeholder:        "/path/to/backup",
					Required:           true,
					Type:               component.InputTypeText,
					HelpText:           "Relative to the base directory of the destination",
					HelpButtonChildren: destinationDirectoryHelp(),
					Pattern:            `^\/\S*[^\/]$`,
					Children: []nodx.Node{
						nodx.Value(backup.DestDir),
					},
				}),

				component.InputControl(component.InputControlParams{
					Name:               "retention_days",
					Label:              "Retention days",
					Placeholder:        "30",
					Required:           true,
					Type:               component.InputTypeNumber,
					Pattern:            "[0-9]+",
					HelpButtonChildren: retentionDaysHelp(),
					Children: []nodx.Node{
						nodx.Min("0"),
						nodx.Max("36500"),
						nodx.Value(fmt.Sprintf("%d", backup.RetentionDays)),
					},
				}),

				component.SelectControl(component.SelectControlParams{
					Name:     "is_active",
					Label:    "Activate backup",
					Required: true,
					Children: []nodx.Node{
						yesNoOptions(backup.IsActive),
					},
				}),

				component.SelectControl(component.SelectControlParams{
					Name:     "is_local",
					Label:    "Local backup",
					Required: true,
					Children: []nodx.Node{
						alpine.XModel("is_local"),
						nodx.Option(
							nodx.Value("true"),
							nodx.Text("Yes"),
							nodx.If(backup.IsLocal, nodx.Selected("")),
						),
						nodx.Option(
							nodx.Value("false"),
							nodx.Text("No"),
							nodx.If(!backup.IsLocal, nodx.Selected("")),
						),
					},
					HelpButtonChildren: localBackupsHelp(),
				}),

				alpine.Template(
					alpine.XIf("is_local == 'false'"),
					component.SelectControl(component.SelectControlParams{
						Name:        "destination_id",
						Label:       "Destination",
						Required:    true,
						Placeholder: "Select a destination",
						Children: []nodx.Node{
							nodx.Map(
								destinations,
								func(dest dbgen.DestinationsServiceGetAllDestinationsRow) nodx.Node {
									return nodx.Option(
										nodx.Value(dest.ID.String()),
										nodx.Text(dest.Name),
										nodx.If(
											backup.DestinationID.Valid && backup.DestinationID.UUID == dest.ID,
											nodx.Selected(""),
										),
									)
								},
							),
						},
					}),
				),

				nodx.Div(
					nodx.Class("pt-4"),
					nodx.Div(
						nodx.Class("flex justify-start items-center space-x-1"),
						component.H2Text("File management"),
					),
					nodx.Div(
						nodx.Class("mt-2 grid grid-cols-2 gap-2"),
						component.SelectControl(component.SelectControlParams{
							Name:     "compression_level",
							Label:    "Compression level",
							Required: false,
							HelpText: "ZIP compression level. Default is best compression",
							Children: []nodx.Node{
								nodx.Option(
									nodx.Value(""),
									nodx.Text("Default (best)"),
									nodx.If(!backup.CompressionLevel.Valid, nodx.Selected("")),
								),
								nodx.Option(
									nodx.Value("9"),
									nodx.Text("Best (9)"),
									nodx.If(backup.CompressionLevel.Valid && backup.CompressionLevel.Int16 == 9, nodx.Selected("")),
								),
								nodx.Option(
									nodx.Value("6"),
									nodx.Text("Balanced (6)"),
									nodx.If(backup.CompressionLevel.Valid && backup.CompressionLevel.Int16 == 6, nodx.Selected("")),
								),
								nodx.Option(
									nodx.Value("1"),
									nodx.Text("Fastest (1)"),
									nodx.If(backup.CompressionLevel.Valid && backup.CompressionLevel.Int16 == 1, nodx.Selected("")),
								),
								nodx.Option(
									nodx.Value("0"),
									nodx.Text("None (store only)"),
									nodx.If(backup.CompressionLevel.Valid && backup.CompressionLevel.Int16 == 0, nodx.Selected("")),
								),
							},
						}),
						component.InputControl(component.InputControlParams{
							Name:        "max_part_size_mb",
							Label:       "Max part size (MB)",
							Placeholder: "Leave empty for single file",
							Required:    false,
							Type:        component.InputTypeNumber,
							HelpText:    "Split backup into parts of this size. Leave empty to keep as a single file",
							Children: []nodx.Node{
								nodx.Min("1"),
								nodx.Max("10000"),
								nodx.If(
									backup.MaxPartSizeMb.Valid,
									nodx.Value(strconv.Itoa(int(backup.MaxPartSizeMb.Int32))),
								),
							},
						}),
					),
				),

				nodx.Div(
					nodx.Class("pt-4"),
					nodx.Div(
						nodx.Class("flex justify-start items-center space-x-1"),
						component.H2Text("Options"),
						component.HelpButtonModal(component.HelpButtonModalParams{
							ModalTitle: "Backup options",
							Children:   pgDumpOptionsHelp(),
						}),
					),

					nodx.Div(
						nodx.Class("mt-2 grid grid-cols-2 gap-2"),
						component.SelectControl(component.SelectControlParams{
							Name:     "opt_data_only",
							Label:    "--data-only",
							Required: true,
							Children: []nodx.Node{
								yesNoOptions(backup.OptDataOnly),
							},
						}),

						component.SelectControl(component.SelectControlParams{
							Name:     "opt_schema_only",
							Label:    "--schema-only",
							Required: true,
							Children: []nodx.Node{
								yesNoOptions(backup.OptSchemaOnly),
							},
						}),

						component.SelectControl(component.SelectControlParams{
							Name:     "opt_clean",
							Label:    "--clean",
							Required: true,
							Children: []nodx.Node{
								yesNoOptions(backup.OptClean),
							},
						}),

						component.SelectControl(component.SelectControlParams{
							Name:     "opt_if_exists",
							Label:    "--if-exists",
							Required: true,
							Children: []nodx.Node{
								yesNoOptions(backup.OptIfExists),
							},
						}),

						component.SelectControl(component.SelectControlParams{
							Name:     "opt_create",
							Label:    "--create",
							Required: true,
							Children: []nodx.Node{
								yesNoOptions(backup.OptCreate),
							},
						}),

						component.SelectControl(component.SelectControlParams{
							Name:     "opt_no_comments",
							Label:    "--no-comments",
							Required: true,
							Children: []nodx.Node{
								yesNoOptions(backup.OptNoComments),
							},
						}),
					),
				),

				nodx.Div(
					nodx.Class("flex justify-end items-center space-x-2 pt-2"),
					component.HxLoadingMd(),
					nodx.Button(
						nodx.Class("btn btn-primary"),
						nodx.Type("submit"),
						component.SpanText("Save"),
						lucide.Save(),
					),
				),
	)
}
