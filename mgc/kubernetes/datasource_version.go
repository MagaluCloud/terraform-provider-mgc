package kubernetes

import (
	"context"

	sdkK8s "github.com/MagaluCloud/mgc-sdk-go/kubernetes"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type VersionsModel struct {
	IncludeDeprecated types.Bool     `tfsdk:"include_deprecated"`
	Versions          []VersionModel `tfsdk:"versions"`
}

type VersionModel struct {
	Deprecated types.Bool   `tfsdk:"deprecated"`
	Version    types.String `tfsdk:"version"`
}

type DataSourceKubernetesVersion struct {
	sdkClient sdkK8s.VersionService
}

func NewDataSourceKubernetesVersion() datasource.DataSource {
	return &DataSourceKubernetesVersion{}
}

func (r *DataSourceKubernetesVersion) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kubernetes_version"
}

func (r *DataSourceKubernetesVersion) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(utils.DataConfig)

	if !ok {
		resp.Diagnostics.AddError("Failed to configure data source", "Invalid provider data")
		return
	}

	r.sdkClient = sdkK8s.New(&dataConfig.CoreConfig).Versions()
}

func (r *DataSourceKubernetesVersion) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"include_deprecated": schema.BoolAttribute{
				Description: "Include deprecated versions.",
				Optional:    true,
			},
			"versions": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of available Kubernetes versions.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"deprecated": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether the version is deprecated.",
						},
						"version": schema.StringAttribute{
							Computed:    true,
							Description: "The Kubernetes version.",
						},
					},
				},
			},
		},
	}
	resp.Schema.Description = "Get the available versions of Kubernetes."
}

func (r *DataSourceKubernetesVersion) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data VersionsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	includeDeprecated := false
	if data.IncludeDeprecated.ValueBoolPointer() != nil && *data.IncludeDeprecated.ValueBoolPointer() {
		includeDeprecated = true
	} else {
		data.IncludeDeprecated = types.BoolValue(false)
	}

	sdkOutput, err := r.sdkClient.List(ctx, includeDeprecated)
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	for _, version := range sdkOutput {
		data.Versions = append(data.Versions, VersionModel{
			Deprecated: types.BoolValue(version.Deprecated),
			Version:    types.StringValue(version.Version),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
