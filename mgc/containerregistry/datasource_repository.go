package containerregistry

import (
	"context"

	crSDK "github.com/MagaluCloud/mgc-sdk-go/containerregistry"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DataSourceCRRepository{}

type DataSourceCRRepository struct {
	crRepositories crSDK.RepositoriesService
}

type crRepositorySingle struct {
	RegistryID   types.String `tfsdk:"registry_id"`
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	RegistryName types.String `tfsdk:"registry_name"`
	ImageCount   types.Int64  `tfsdk:"image_count"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

func NewDataSourceCRRepository() datasource.DataSource {
	return &DataSourceCRRepository{}
}

func (r *DataSourceCRRepository) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_repository"
}

func (r *DataSourceCRRepository) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *DataSourceCRRepository) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a single repository from a Container Registry by its UUID.",
		Attributes: map[string]schema.Attribute{
			"registry_id": schema.StringAttribute{
				Description: "ID of the registry containing the repository",
				Required:    true,
			},
			"id": schema.StringAttribute{
				Description: "Unique identifier (UUID) of the repository",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the repository",
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
	}
}

func (r *DataSourceCRRepository) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data crRepositorySingle
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repository, err := r.crRepositories.GetByID(ctx, data.RegistryID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data.ID = types.StringValue(repository.ID)
	data.Name = types.StringValue(repository.Name)
	data.RegistryName = types.StringValue(repository.RegistryName)
	data.ImageCount = types.Int64Value(int64(repository.ImageCount))
	data.CreatedAt = types.StringValue(repository.CreatedAt)
	data.UpdatedAt = types.StringValue(repository.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
