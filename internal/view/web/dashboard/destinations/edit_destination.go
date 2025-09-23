package destinations

import (
	"database/sql"

	"github.com/eduardolat/pgbackweb/internal/database/dbgen"
	"github.com/eduardolat/pgbackweb/internal/validate"
	"github.com/eduardolat/pgbackweb/internal/view/web/component"
	"github.com/eduardolat/pgbackweb/internal/view/web/respondhtmx"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	htmx "github.com/nodxdev/nodxgo-htmx"
	lucide "github.com/nodxdev/nodxgo-lucide"
)

func (h *handlers) editDestinationHandler(c echo.Context) error {
	ctx := c.Request().Context()

	destinationID, err := uuid.Parse(c.Param("destinationID"))
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	var formData createDestinationDTO
	if err := c.Bind(&formData); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}
	if err := validate.Struct(&formData); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	_, err = h.servs.DestinationsService.UpdateDestination(
			ctx, dbgen.DestinationsServiceUpdateDestinationParams{
				ID:             destinationID,
				Name:           sql.NullString{String: formData.Name, Valid: true},
				BucketName:     sql.NullString{String: formData.BucketName, Valid: true},
				Region:         sql.NullString{String: formData.Region, Valid: true},
				Endpoint:       sql.NullString{String: formData.Endpoint, Valid: true},
				Provider:       sql.NullString{String: formData.Provider, Valid: true},
				ForcePathStyle: sql.NullBool{Bool: formData.ForcePathStyle, Valid: true},
				AccessKey:      sql.NullString{String: formData.AccessKey, Valid: true},
				SecretKey:      sql.NullString{String: formData.SecretKey, Valid: true},
			},
		)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	return respondhtmx.AlertWithRefresh(c, "Destination updated")
}

func editDestinationButton(
	destination dbgen.DestinationsServicePaginateDestinationsRow,
) nodx.Node {
	idPref := "edit-destination-" + destination.ID.String()
	formID := idPref + "-form"
	btnClass := idPref + "-btn"
	loadingID := idPref + "-loading"

	htmxAttributes := func(url string) nodx.Node {
		return nodx.Group(
			htmx.HxPost(url),
			htmx.HxInclude("#"+formID),
			htmx.HxDisabledELT("."+btnClass),
			htmx.HxIndicator("#"+loadingID),
			htmx.HxValidate("true"),
		)
	}

	mo := component.Modal(component.ModalParams{
		Size:  component.SizeMd,
		Title: "Edit destination",
		Content: []nodx.Node{
			nodx.FormEl(
				nodx.Id(formID),
				nodx.Class("space-y-2"),

				component.InputControl(component.InputControlParams{
					Name:        "name",
					Label:       "Name",
					Placeholder: "My destination",
					Required:    true,
					Type:        component.InputTypeText,
					HelpText:    "A name to easily identify the destination",
					Children: []nodx.Node{
						nodx.Value(destination.Name),
					},
				}),

				component.InputControl(component.InputControlParams{
					Name:        "bucket_name",
					Label:       "Bucket name",
					Placeholder: "my-bucket",
					Required:    true,
					Type:        component.InputTypeText,
					Children: []nodx.Node{
						nodx.Value(destination.BucketName),
					},
				}),

				component.InputControl(component.InputControlParams{
					Name:        "endpoint",
					Label:       "Endpoint",
					Placeholder: "s3-us-west-1.amazonaws.com",
					Required:    true,
					Type:        component.InputTypeText,
					Children: []nodx.Node{
						nodx.Value(destination.Endpoint),
					},
				}),

				component.InputControl(component.InputControlParams{
					Name:        "region",
					Label:       "Region",
					Placeholder: "us-west-1",
					Required:    true,
					Type:        component.InputTypeText,
					Children: []nodx.Node{
						nodx.Value(destination.Region),
					},
				}),

				component.InputControl(component.InputControlParams{
					Name:        "access_key",
					Label:       "Access key",
					Placeholder: "Access key",
					Required:    true,
					Type:        component.InputTypeText,
					HelpText:    "It will be stored securely using PGP encryption.",
					Children: []nodx.Node{
						nodx.Value(destination.DecryptedAccessKey),
					},
				}),

				component.InputControl(component.InputControlParams{
					Name:        "secret_key",
					Label:       "Secret key",
					Placeholder: "Secret key",
					Required:    true,
					Type:        component.InputTypeText,
					HelpText:    "It will be stored securely using PGP encryption.",
					Children: []nodx.Node{
						nodx.Value(destination.DecryptedSecretKey),
					},
				}),

				component.SelectControl(component.SelectControlParams{
					Name:        "provider",
					Label:       "Provider",
					Required:    true,
					Placeholder: "Select provider",
					HelpText:    "Choose between AWS S3 or MinIO compatible storage",
					Children: []nodx.Node{
						nodx.Option(
							nodx.Value("aws"), 
							nodx.Text("AWS S3"),
							nodx.If(destination.Provider == "aws", nodx.Selected("")),
						),
						nodx.Option(
							nodx.Value("minio"), 
							nodx.Text("MinIO"),
							nodx.If(destination.Provider == "minio", nodx.Selected("")),
						),
					},
				}),

				nodx.Div(
					nodx.Class("form-control"),
					nodx.LabelEl(
						nodx.Class("cursor-pointer label justify-start space-x-2"),
						nodx.Input(
							nodx.Type("checkbox"),
							nodx.Name("force_path_style"),
							nodx.Class("checkbox"),
							nodx.If(destination.ForcePathStyle, nodx.Checked("")),
						),
						nodx.SpanEl(
							nodx.Class("label-text"),
							nodx.Text("Force path style"),
						),
					),
					nodx.Div(
						nodx.Class("label"),
						nodx.SpanEl(
							nodx.Class("label-text-alt text-xs"),
							nodx.Text("Enable for MinIO or S3-compatible services that require path-style URLs"),
						),
					),
				),
			),

			nodx.Div(
				nodx.Class("flex justify-between items-center pt-4"),
				nodx.Div(
					nodx.Button(
						htmxAttributes("/dashboard/destinations/test"),
						nodx.ClassMap{
							btnClass:                      true,
							"btn btn-neutral btn-outline": true,
						},
						nodx.Type("button"),
						component.SpanText("Test connection"),
						lucide.PlugZap(),
					),
				),
				nodx.Div(
					nodx.Class("flex justify-end items-center space-x-2"),
					component.HxLoadingMd(loadingID),
					nodx.Button(
						htmxAttributes("/dashboard/destinations/"+destination.ID.String()+"/edit"),
						nodx.ClassMap{
							btnClass:          true,
							"btn btn-primary": true,
						},
						nodx.Type("button"),
						component.SpanText("Save"),
						lucide.Save(),
					),
				),
			),
		},
	})

	return nodx.Div(
		mo.HTML,
		component.OptionsDropdownButton(
			mo.OpenerAttr,
			lucide.Pencil(),
			component.SpanText("Edit destination"),
		),
	)
}
