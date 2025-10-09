package containerregistry

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	crSDK "github.com/MagaluCloud/mgc-sdk-go/containerregistry"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DataSourceCRRepositories{}

type DataSourceCRRepositories struct {
	crRepositories crSDK.RepositoriesService
}

func NewDataSourceCRRepositories() datasource.DataSource {
	return &DataSourceCRRepositories{}
}

func (r *DataSourceCRRepositories) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_repositories"
}

type crRepository struct {
	Name         types.String `tfsdk:"name"`
	RegistryName types.String `tfsdk:"registry_name"`
	ImageCount   types.Int64  `tfsdk:"image_count"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

type crRepositoriesList struct {
	RegistryID   types.String   `tfsdk:"registry_id"`
	Repositories []crRepository `tfsdk:"repositories"`
}

func (r *DataSourceCRRepositories) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to configure data source", "Invalid provider data")
		return
	}

	r.crRepositories = crSDK.New(&dataConfig.CoreConfig).Repositories()
}
func (r *DataSourceCRRepositories) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data source for Container Registry Repositories",
		Attributes: map[string]schema.Attribute{
			"registry_id": schema.StringAttribute{
				Description: "ID of the registry",
				Required:    true,
			},
			"repositories": schema.ListNestedAttribute{
				Description: "List of container repositories",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The name of the repository",
							Computed:    true,
						},
						"registry_name": schema.StringAttribute{
							Description: "The name of the registry containing this repository",
							Computed:    true,
						},
						"image_count": schema.Int64Attribute{
							Description: "The number of images in the repository",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "The timestamp when the repository was created",
							Computed:    true,
						},
						"updated_at": schema.StringAttribute{
							Description: "The timestamp when the repository was last updated",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (r *DataSourceCRRepositories) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data crRepositoriesList

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sdkOutputList, err := r.getAllRepositories(ctx, data.RegistryID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	for _, sdkOutput := range sdkOutputList {
		var item crRepository
		item.CreatedAt = types.StringValue(sdkOutput.CreatedAt)
		item.ImageCount = types.Int64Value(int64(sdkOutput.ImageCount))
		item.RegistryName = types.StringValue(sdkOutput.RegistryName)
		item.UpdatedAt = types.StringValue(sdkOutput.UpdatedAt)
		item.Name = types.StringValue(sdkOutput.Name)

		data.Repositories = append(data.Repositories, item)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DataSourceCRRepositories) getAllRepositories(ctx context.Context, registryID string) ([]crSDK.RepositoryResponse, error) {
	params := crSDK.ListOptions{
		Limit:  &listLimit,
		Offset: new(int),
	}

	var allRegistries []crSDK.RepositoryResponse
	for {
		response, err := r.crRepositories.List(ctx, registryID, params)
		if err != nil {
			return nil, err
		}

		allRegistries = append(allRegistries, response.Results...)
		if len(allRegistries) == response.Meta.Page.Total {
			return allRegistries, nil
		}
		*params.Offset = response.Meta.Page.Offset + response.Meta.Page.Count
	}
}
