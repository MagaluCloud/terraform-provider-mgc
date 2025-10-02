package containerregistry

import (
	"context"
	"regexp"

	crSDK "github.com/MagaluCloud/mgc-sdk-go/containerregistry"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DataSourceCRRegistries{}

type DataSourceCRRegistries struct {
	crRegistries crSDK.RegistriesService
}

func NewDataSourceCRRegistries() datasource.DataSource {
	return &DataSourceCRRegistries{}
}

func (r *DataSourceCRRegistries) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_registries"
}

type crRegistries struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	StorageUsageBytes types.Int64  `tfsdk:"storage_usage_bytes"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}

type crRegistriesList struct {
	Registries []crRegistries    `tfsdk:"registries"`
	Options    utils.ListOptions `tfsdk:"options"`
}

func (r *DataSourceCRRegistries) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to configure data source", "Invalid provider data")
		return
	}

	r.crRegistries = crSDK.New(&dataConfig.CoreConfig).Registries()
}

func (r *DataSourceCRRegistries) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data source for Container Registry lists",
		Attributes: map[string]schema.Attribute{
			"registries": schema.ListNestedAttribute{
				Description: "List of container registries",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique identifier of the registry",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the registry",
							Computed:    true,
						},
						"storage_usage_bytes": schema.Int64Attribute{
							Description: "The storage usage in bytes",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "The timestamp when the registry was created",
							Computed:    true,
						},
						"updated_at": schema.StringAttribute{
							Description: "The timestamp when the registry was last updated",
							Computed:    true,
						},
					},
				},
			},
			"options": schema.SingleNestedAttribute{
				Description: "Options for listing registries",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"limit": schema.Int64Attribute{
						Description: "Maximum number of registries to return",
						Validators: []validator.Int64{
							int64validator.AtLeast(1),
						},
						Optional: true,
					},
					"offset": schema.Int64Attribute{
						Description: "Number of registries to skip before starting to collect the result set",
						Validators: []validator.Int64{
							int64validator.AtLeast(0),
						},
						Optional: true,
					},
					"sort": schema.StringAttribute{
						Description: "Field by which to sort the registries",
						Validators: []validator.String{
							stringvalidator.RegexMatches(regexp.MustCompile(`(^[\w-]+:(asc|desc)(,[\w-]+:(asc|desc))*)?$`), "Must be in the format 'field:direction', e.g., 'created_at:asc'"),
						},
						Optional: true,
					},
					// NOTE: OpenAPI spect doesn't define "expand", but structure from SDK does; what to do?
					"expand": schema.ListAttribute{
						Description: "List of related resources to expand in the response",
						Optional:    true,
						ElementType: types.StringType,
					},
				},
			},
		},
	}
}

func (r *DataSourceCRRegistries) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data crRegistriesList

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	opts := crSDK.ListOptions{}

	diag := utils.ConvertListOptions(data.Options, &opts)
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	sdkOutputList, err := r.crRegistries.List(ctx, opts)
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	for _, sdkOutput := range sdkOutputList.Registries {

		var item crRegistries

		item.ID = types.StringValue(sdkOutput.ID)
		item.Name = types.StringValue(sdkOutput.Name)
		item.UpdatedAt = types.StringValue(sdkOutput.UpdatedAt)
		item.CreatedAt = types.StringValue(sdkOutput.CreatedAt)

		data.Registries = append(data.Registries, item)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
