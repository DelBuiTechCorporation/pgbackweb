package destinations

import (
	"github.com/eduardolat/pgbackweb/internal/database/dbgen"
	"github.com/eduardolat/pgbackweb/internal/validate"
	"github.com/eduardolat/pgbackweb/internal/view/web/component"
	"github.com/eduardolat/pgbackweb/internal/view/web/respondhtmx"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	htmx "github.com/nodxdev/nodxgo-htmx"
	lucide "github.com/nodxdev/nodxgo-lucide"
)

type createDestinationDTO struct {
	Name           string `form:"name" validate:"required"`
	BucketName     string `form:"bucket_name" validate:"required"`
	AccessKey      string `form:"access_key" validate:"required"`
	SecretKey      string `form:"secret_key" validate:"required"`
	Region         string `form:"region" validate:"required"`
	Endpoint       string `form:"endpoint" validate:"required"`
	Provider       string `form:"provider" validate:"required,oneof=minio aws"`
	ForcePathStyle bool   `form:"force_path_style"`
}

func (h *handlers) createDestinationHandler(c echo.Context) error {
	ctx := c.Request().Context()

	var formData createDestinationDTO
	if err := c.Bind(&formData); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}
	if err := validate.Struct(&formData); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	_, err := h.servs.DestinationsService.CreateDestination(
		ctx, dbgen.DestinationsServiceCreateDestinationParams{
			Name:           formData.Name,
			AccessKey:      formData.AccessKey,
			SecretKey:      formData.SecretKey,
			Region:         formData.Region,
			Endpoint:       formData.Endpoint,
			BucketName:     formData.BucketName,
			Provider:       formData.Provider,
			ForcePathStyle: formData.ForcePathStyle,
		},
	)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	return respondhtmx.Redirect(c, "/dashboard/destinations")
}

func createDestinationButton() nodx.Node {
	htmxAttributes := func(url string) nodx.Node {
		return nodx.Group(
			htmx.HxPost(url),
			htmx.HxInclude("#add-destination-form"),
			htmx.HxDisabledELT(".add-destination-btn"),
			htmx.HxIndicator("#add-destination-loading"),
			htmx.HxValidate("true"),
		)
	}

	mo := component.Modal(component.ModalParams{
		Size:  component.SizeMd,
		Title: "Add destination",
		Content: []nodx.Node{
			nodx.FormEl(
				nodx.Id("add-destination-form"),
				nodx.Class("space-y-2"),

				component.InputControl(component.InputControlParams{
					Name:        "name",
					Label:       "Name",
					Placeholder: "My destination",
					Required:    true,
					Type:        component.InputTypeText,
					HelpText:    "A name to easily identify the destination",
				}),

				component.InputControl(component.InputControlParams{
					Name:        "bucket_name",
					Label:       "Bucket name",
					Placeholder: "my-bucket",
					Required:    true,
					Type:        component.InputTypeText,
				}),

				component.InputControl(component.InputControlParams{
					Name:        "endpoint",
					Label:       "Endpoint",
					Placeholder: "s3-us-west-1.amazonaws.com",
					Required:    true,
					Type:        component.InputTypeText,
				}),

				component.InputControl(component.InputControlParams{
					Name:        "region",
					Label:       "Region",
					Placeholder: "us-west-1",
					Required:    true,
					Type:        component.InputTypeText,
				}),

				component.InputControl(component.InputControlParams{
					Name:        "access_key",
					Label:       "Access key",
					Placeholder: "Access key",
					Required:    true,
					Type:        component.InputTypeText,
					HelpText:    "It will be stored securely using PGP encryption.",
				}),

				component.InputControl(component.InputControlParams{
					Name:        "secret_key",
					Label:       "Secret key",
					Placeholder: "Secret key",
					Required:    true,
					Type:        component.InputTypeText,
					HelpText:    "It will be stored securely using PGP encryption.",
				}),

				component.SelectControl(component.SelectControlParams{
					Name:        "provider",
					Label:       "Provider",
					Required:    true,
					Placeholder: "Select provider",
					HelpText:    "Choose between AWS S3 or MinIO compatible storage",
					Children: []nodx.Node{
						nodx.Option(nodx.Value("aws"), nodx.Text("AWS S3")),
						nodx.Option(nodx.Value("minio"), nodx.Text("MinIO")),
					},
				}),

				component.SelectControl(component.SelectControlParams{
					Name:        "force_path_style",
					Label:       "Force path style",
					Required:    true,
					Placeholder: "Select option",
					HelpText:    "Enable for MinIO or S3-compatible services that require path-style URLs",
					Children: []nodx.Node{
						nodx.Option(nodx.Value("false"), nodx.Text("No")),
						nodx.Option(nodx.Value("true"), nodx.Text("Yes")),
					},
				}),
			),

			nodx.Div(
				nodx.Class("flex justify-between items-center pt-4"),
				nodx.Div(
					nodx.Button(
						htmxAttributes("/dashboard/destinations/test"),
						nodx.Class("add-destination-btn btn btn-neutral btn-outline"),
						nodx.Type("button"),
						component.SpanText("Test connection"),
						lucide.PlugZap(),
					),
				),
				nodx.Div(
					nodx.Class("flex justify-end items-center space-x-2"),
					component.HxLoadingMd("add-destination-loading"),
					nodx.Button(
						htmxAttributes("/dashboard/destinations"),
						nodx.Class("add-destination-btn btn btn-primary"),
						nodx.Type("button"),
						component.SpanText("Add destination"),
						lucide.Save(),
					),
				),
			),
		},
	})

	button := nodx.Button(
		mo.OpenerAttr,
		nodx.Class("btn btn-primary"),
		component.SpanText("Add destination"),
		lucide.Plus(),
	)

	return nodx.Div(
		nodx.Class("inline-block"),
		mo.HTML,
		button,
	)
}
